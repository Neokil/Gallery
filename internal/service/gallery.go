package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"image/gif"
	_ "image/gif" // Register GIF format
	"image/jpeg"
	_ "image/jpeg" // Register JPEG format
	"image/png"
	_ "image/png" // Register PNG format

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

const (
	filePermissions  = 0600 // File permissions for metadata files
	thumbnailSize    = 300  // Thumbnail max width/height in pixels
	thumbnailQuality = 80   // JPEG quality for thumbnails (0-100)
)

type PhotoInfo struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Uploader  string    `json:"uploader"`
	Event     string    `json:"event"`
	Date      time.Time `json:"date"`      // Upload/file modification time
	PhotoTime time.Time `json:"photo_time"` // Actual photo taken time from EXIF
}

// dateWalker implements exif.Walker to find date fields in EXIF data
type dateWalker struct {
	foundDate time.Time
}

func (w *dateWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	if !w.foundDate.IsZero() {
		return nil // Already found a date
	}

	// Check if field name suggests it contains date/time
	fieldStr := string(name)
	if strings.Contains(strings.ToLower(fieldStr), "date") ||
		strings.Contains(strings.ToLower(fieldStr), "time") {

		if dateStr, err := tag.StringVal(); err == nil {
			log.Printf("Found potential date field %s: %s", name, dateStr)

			// Try various date formats
			dateFormats := []string{
				"2006:01:02 15:04:05",
				"2006-01-02 15:04:05",
				"2006:01:02T15:04:05",
				"2006-01-02T15:04:05",
				"2006:01:02 15:04:05-07:00",
				"2006-01-02 15:04:05-07:00",
				"2006:01:02",
				"2006-01-02",
				"2006/01/02 15:04:05",
				"2006/01/02",
			}

			for _, format := range dateFormats {
				if photoTime, err := time.Parse(format, dateStr); err == nil {
					log.Printf("Successfully parsed date from field %s: %s", name, photoTime.Format(time.RFC3339))
					w.foundDate = photoTime
					return nil
				}
			}
		}
	}
	return nil
}

type GalleryService struct {
	uploadDir    string
	metadataDir  string
	thumbnailDir string
}

func NewGalleryService(uploadDir, metadataDir string) *GalleryService {
	thumbnailDir := filepath.Join(metadataDir, "thumbnails")

	service := &GalleryService{
		uploadDir:    uploadDir,
		metadataDir:  metadataDir,
		thumbnailDir: thumbnailDir,
	}

	// Generate metadata and thumbnails for existing images on startup
	service.GenerateMissingMetadata()
	service.GenerateMissingThumbnails()

	// Clean up orphaned files on startup
	service.CleanupOrphanedMetadata()
	service.CleanupOrphanedThumbnails()

	return service
}

func (s *GalleryService) GetPhotos() ([]PhotoInfo, error) {
	var photos []PhotoInfo

	files, err := os.ReadDir(s.uploadDir)
	if err != nil {
		return photos, err
	}

	for _, file := range files {
		if !file.IsDir() && s.isImageFile(file.Name()) {
			photoInfo := s.loadPhotoMetadata(file.Name())
			if photoInfo.Path == "" {
				// Fallback for photos without metadata - extract photo time
				filePath := filepath.Join(s.uploadDir, file.Name())
				photoTime := s.extractPhotoTime(filePath)

				photoInfo = PhotoInfo{
					Path:      "/uploads/" + file.Name(),
					Name:      file.Name(),
					Uploader:  "Unknown",
					Event:     "",
					Date:      time.Now(),
					PhotoTime: photoTime,
				}
			}
			photos = append(photos, photoInfo)
		}
	}

	// Sort photos by photo taken time (newest first), fall back to upload time if no photo time
	for i := 0; i < len(photos)-1; i++ {
		for j := i + 1; j < len(photos); j++ {
			timeI := photos[i].PhotoTime
			timeJ := photos[j].PhotoTime

			// Use upload time if photo time is not available
			if timeI.IsZero() {
				timeI = photos[i].Date
			}
			if timeJ.IsZero() {
				timeJ = photos[j].Date
			}

			// Sort newest first
			if timeI.Before(timeJ) {
				photos[i], photos[j] = photos[j], photos[i]
			}
		}
	}

	return photos, nil
}

func (s *GalleryService) FilterPhotos(photos []PhotoInfo, eventFilter, uploaderFilter string) []PhotoInfo {
	var filtered []PhotoInfo

	for _, photo := range photos {
		// Check event filter
		if eventFilter != "" && photo.Event != eventFilter {
			continue
		}

		// Check uploader filter
		if uploaderFilter != "" && photo.Uploader != uploaderFilter {
			continue
		}

		filtered = append(filtered, photo)
	}

	return filtered
}

// getUniqueValues is a helper function to extract unique non-empty values from photos
func (s *GalleryService) getUniqueValues(photos []PhotoInfo, extractor func(PhotoInfo) string) []string {
	valueSet := make(map[string]bool)
	var values []string

	for _, photo := range photos {
		value := extractor(photo)
		if value != "" && !valueSet[value] {
			valueSet[value] = true
			values = append(values, value)
		}
	}

	// Sort values alphabetically using bubble sort
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	return values
}

func (s *GalleryService) GetUniqueEvents(photos []PhotoInfo) []string {
	return s.getUniqueValues(photos, func(p PhotoInfo) string { return p.Event })
}

func (s *GalleryService) GetUniqueUploaders(photos []PhotoInfo) []string {
	return s.getUniqueValues(photos, func(p PhotoInfo) string { return p.Uploader })
}

func (s *GalleryService) SavePhoto(fileHeader *multipart.FileHeader, userName, eventName string) error {
	if !s.isValidImageType(fileHeader.Header.Get("Content-Type")) {
		return fmt.Errorf("invalid image type")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Failed to close file: %v", closeErr)
		}
	}()

	// Generate unique filename preserving original name
	filename := s.generateUniqueFilename(fileHeader.Filename)
	filePath := filepath.Join(s.uploadDir, filename)

	// #nosec G304 - filePath is constructed from controlled uploadDir and sanitized filename
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			log.Printf("Failed to close destination file: %v", closeErr)
		}
	}()

	_, err = io.Copy(dst, file)
	if err != nil {
		return err
	}

	// Generate thumbnail
	thumbnailPath := filepath.Join(s.thumbnailDir, filename)
	if err := s.generateThumbnail(filePath, thumbnailPath); err != nil {
		log.Printf("Failed to generate thumbnail for %s: %v", filename, err)
		// Don't fail the upload if thumbnail generation fails
	}

	// Extract photo taken time from EXIF
	photoTime := s.extractPhotoTime(filePath)

	// Save photo metadata
	photoInfo := PhotoInfo{
		Path:      "/uploads/" + filename,
		Name:      filename,
		Uploader:  userName,
		Event:     eventName,
		Date:      time.Now(),
		PhotoTime: photoTime,
	}
	s.savePhotoMetadata(filename, &photoInfo)

	return nil
}

func (s *GalleryService) CreateZipArchive(photos []PhotoInfo, writer io.Writer) error {
	zipWriter := zip.NewWriter(writer)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Printf("Failed to close zip writer: %v", err)
		}
	}()

	for _, photo := range photos {
		filename := filepath.Base(photo.Path)
		filePath := filepath.Join(s.uploadDir, filename)

		// #nosec G304 - filePath is constructed from controlled uploadDir and photo.Name
		fileReader, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open file %s: %v", filename, err)
			continue
		}

		zipFile, err := zipWriter.Create(filename)
		if err != nil {
			log.Printf("Failed to create zip entry for %s: %v", filename, err)
			if closeErr := fileReader.Close(); closeErr != nil {
				log.Printf("Failed to close file reader: %v", closeErr)
			}
			continue
		}

		_, err = io.Copy(zipFile, fileReader)
		if err != nil {
			log.Printf("Failed to copy file %s to zip: %v", filename, err)
		}

		if closeErr := fileReader.Close(); closeErr != nil {
			log.Printf("Failed to close file reader: %v", closeErr)
		}
	}

	return nil
}

func (s *GalleryService) ServePhoto(filename string) (string, error) {
	filePath := filepath.Join(s.uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found")
	}
	return filePath, nil
}

func (s *GalleryService) CleanupOrphanedMetadata() {
	metadataFiles, err := os.ReadDir(s.metadataDir)
	if err != nil {
		log.Printf("Failed to read metadata directory: %v", err)
		return
	}

	removedCount := 0
	for _, metadataFile := range metadataFiles {
		if metadataFile.IsDir() || !strings.HasSuffix(metadataFile.Name(), ".json") {
			continue
		}

		imageFilename := strings.TrimSuffix(metadataFile.Name(), ".json")
		imagePath := filepath.Join(s.uploadDir, imageFilename)

		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			metadataPath := filepath.Join(s.metadataDir, metadataFile.Name())
			if err := os.Remove(metadataPath); err != nil {
				log.Printf("Failed to remove orphaned metadata file %s: %v", metadataFile.Name(), err)
			} else {
				log.Printf("Removed orphaned metadata file: %s", metadataFile.Name())
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		log.Printf("Cleanup complete: removed %d orphaned metadata files", removedCount)
	}
}

func (s *GalleryService) CleanupOrphanedThumbnails() {
	thumbnailFiles, err := os.ReadDir(s.thumbnailDir)
	if err != nil {
		log.Printf("Failed to read thumbnail directory: %v", err)
		return
	}

	removedCount := 0
	for _, thumbnailFile := range thumbnailFiles {
		if thumbnailFile.IsDir() || !s.isImageFile(thumbnailFile.Name()) {
			continue
		}

		// Check if corresponding original image exists
		originalImagePath := filepath.Join(s.uploadDir, thumbnailFile.Name())
		if _, err := os.Stat(originalImagePath); os.IsNotExist(err) {
			thumbnailPath := filepath.Join(s.thumbnailDir, thumbnailFile.Name())
			if err := os.Remove(thumbnailPath); err != nil {
				log.Printf("Failed to remove orphaned thumbnail file %s: %v", thumbnailFile.Name(), err)
			} else {
				log.Printf("Removed orphaned thumbnail file: %s", thumbnailFile.Name())
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		log.Printf("Thumbnail cleanup complete: removed %d orphaned thumbnail files", removedCount)
	}
}

func (s *GalleryService) GenerateMissingMetadata() {
	// Ensure directories exist
	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		log.Printf("Failed to create upload directory: %v", err)
		return
	}
	if err := os.MkdirAll(s.metadataDir, 0755); err != nil {
		log.Printf("Failed to create metadata directory: %v", err)
		return
	}

	files, err := os.ReadDir(s.uploadDir)
	if err != nil {
		log.Printf("Failed to read upload directory: %v", err)
		return
	}

	generatedCount := 0
	for _, file := range files {
		if file.IsDir() || !s.isImageFile(file.Name()) {
			continue
		}

		// Check if metadata already exists
		metadataFile := filepath.Join(s.metadataDir, file.Name()+".json")
		if _, err := os.Stat(metadataFile); err == nil {
			continue // Metadata already exists
		}

		// Get file info for creation date
		fileInfo, err := file.Info()
		if err != nil {
			log.Printf("Failed to get file info for %s: %v", file.Name(), err)
			continue
		}

		// Extract photo taken time from EXIF
		filePath := filepath.Join(s.uploadDir, file.Name())
		photoTime := s.extractPhotoTime(filePath)

		// Generate default metadata
		photoInfo := PhotoInfo{
			Path:      "/uploads/" + file.Name(),
			Name:      file.Name(),
			Uploader:  "Unknown",
			Event:     "",
			Date:      fileInfo.ModTime(),
			PhotoTime: photoTime,
		}

		// Save the generated metadata
		s.savePhotoMetadata(file.Name(), &photoInfo)
		generatedCount++
		log.Printf("Generated metadata for existing image: %s", file.Name())
	}

	if generatedCount > 0 {
		log.Printf("Startup metadata generation complete: created %d metadata files", generatedCount)
	} else {
		log.Printf("All existing images already have metadata")
	}
}

func (s *GalleryService) GenerateMissingThumbnails() {
	// Ensure thumbnail directory exists
	if err := os.MkdirAll(s.thumbnailDir, 0755); err != nil {
		log.Printf("Failed to create thumbnail directory: %v", err)
		return
	}

	files, err := os.ReadDir(s.uploadDir)
	if err != nil {
		log.Printf("Failed to read upload directory for thumbnails: %v", err)
		return
	}

	generatedCount := 0
	for _, file := range files {
		if file.IsDir() || !s.isImageFile(file.Name()) {
			continue
		}

		// Check if thumbnail already exists
		thumbnailPath := filepath.Join(s.thumbnailDir, file.Name())
		if _, err := os.Stat(thumbnailPath); err == nil {
			continue // Thumbnail already exists
		}

		// Generate thumbnail
		originalPath := filepath.Join(s.uploadDir, file.Name())
		if err := s.generateThumbnail(originalPath, thumbnailPath); err != nil {
			log.Printf("Failed to generate thumbnail for %s: %v", file.Name(), err)
			continue
		}

		generatedCount++
		log.Printf("Generated thumbnail for existing image: %s", file.Name())
	}

	if generatedCount > 0 {
		log.Printf("Startup thumbnail generation complete: created %d thumbnails", generatedCount)
	} else {
		log.Printf("All existing images already have thumbnails")
	}
}

func (s *GalleryService) generateThumbnail(originalPath, thumbnailPath string) error {
	// Open original image
	originalFile, err := os.Open(originalPath)
	if err != nil {
		return fmt.Errorf("failed to open original image: %w", err)
	}
	defer originalFile.Close()

	// Decode image
	img, format, err := image.Decode(originalFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate thumbnail dimensions maintaining aspect ratio
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var newWidth, newHeight int
	if width > height {
		newWidth = thumbnailSize
		newHeight = (height * thumbnailSize) / width
	} else {
		newHeight = thumbnailSize
		newWidth = (width * thumbnailSize) / height
	}

	// Create thumbnail using simple nearest neighbor scaling
	thumbnail := s.resizeImage(img, newWidth, newHeight)

	// Create thumbnail file
	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer thumbnailFile.Close()

	// Encode thumbnail based on original format
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(thumbnailFile, thumbnail, &jpeg.Options{Quality: thumbnailQuality})
	case "png":
		err = png.Encode(thumbnailFile, thumbnail)
	case "gif":
		err = gif.Encode(thumbnailFile, thumbnail, nil)
	default:
		// Default to JPEG for unknown formats
		err = jpeg.Encode(thumbnailFile, thumbnail, &jpeg.Options{Quality: thumbnailQuality})
	}

	if err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return nil
}

// Simple image resizing using nearest neighbor
func (s *GalleryService) resizeImage(src image.Image, width, height int) image.Image {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := (x * srcWidth) / width
			srcY := (y * srcHeight) / height
			dst.Set(x, y, src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}

	return dst
}

func (s *GalleryService) ServeThumbnail(filename string) (string, error) {
	thumbnailPath := filepath.Join(s.thumbnailDir, filename)
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		return "", fmt.Errorf("thumbnail not found")
	}
	return thumbnailPath, nil
}

// Private helper methods

func (s *GalleryService) isValidImageType(contentType string) bool {
	validTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"}
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func (s *GalleryService) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func (s *GalleryService) generateUniqueFilename(originalFilename string) string {
	originalFilename = filepath.Base(originalFilename)

	filePath := filepath.Join(s.uploadDir, originalFilename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return originalFilename
	}

	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)

	counter := 1
	for {
		newFilename := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		filePath := filepath.Join(s.uploadDir, newFilename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return newFilename
		}
		counter++
	}
}

func (s *GalleryService) savePhotoMetadata(filename string, info *PhotoInfo) {
	metadataFile := filepath.Join(s.metadataDir, filename+".json")
	data, err := json.Marshal(info)
	if err != nil {
		log.Printf("Failed to marshal metadata for %s: %v", filename, err)
		return
	}

	err = os.WriteFile(metadataFile, data, filePermissions)
	if err != nil {
		log.Printf("Failed to save metadata for %s: %v", filename, err)
	}
}

func (s *GalleryService) loadPhotoMetadata(filename string) PhotoInfo {
	metadataFile := filepath.Join(s.metadataDir, filename+".json")
	// #nosec G304 - metadataFile is constructed from controlled metadataDir and filename
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return PhotoInfo{}
	}

	var info PhotoInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		log.Printf("Failed to unmarshal metadata for %s: %v", filename, err)
		return PhotoInfo{}
	}

	return info
}

func (s *GalleryService) extractPhotoTime(filePath string) time.Time {
	// First try exiftool for comprehensive metadata extraction
	if photoTime := s.extractPhotoTimeWithExifTool(filePath); !photoTime.IsZero() {
		return photoTime
	}

	// Fallback to Go EXIF library for basic EXIF data
	if photoTime := s.extractExifPhotoTime(filePath); !photoTime.IsZero() {
		return photoTime
	}

	// No date found from EXIF metadata
	return time.Time{}
}

func (s *GalleryService) extractExifPhotoTime(filePath string) time.Time {
	file, err := os.Open(filePath)
	if err != nil {
		return time.Time{} // Return zero time if can't open file
	}
	defer file.Close()

	// Try to decode EXIF data
	exifData, err := exif.Decode(file)
	if err != nil {
		// Not an error - many image formats don't have EXIF
		return time.Time{} // Return zero time if no EXIF data
	}

	// Try multiple EXIF date fields in order of preference
	dateFields := []exif.FieldName{
		exif.DateTimeOriginal,  // When photo was taken (preferred)
		exif.DateTime,          // When photo was last modified
		exif.DateTimeDigitized, // When photo was digitized
	}

	for _, field := range dateFields {
		if tag, err := exifData.Get(field); err == nil {
			if dateStr, err := tag.StringVal(); err == nil {
				// Try multiple date formats
				dateFormats := []string{
					"2006:01:02 15:04:05",       // Standard EXIF format
					"2006-01-02 15:04:05",       // Alternative format
					"2006:01:02T15:04:05",       // ISO-like with colons
					"2006-01-02T15:04:05",       // ISO format
					"2006:01:02 15:04:05-07:00", // With timezone
					"2006-01-02 15:04:05-07:00", // With timezone
				}

				for _, format := range dateFormats {
					if photoTime, err := time.Parse(format, dateStr); err == nil {
						log.Printf("Extracted photo time from EXIF %s for %s: %s (format: %s)", field, filepath.Base(filePath), photoTime.Format(time.RFC3339), format)
						return photoTime
					}
				}

				log.Printf("Found EXIF %s for %s but couldn't parse date: %s", field, filepath.Base(filePath), dateStr)
			}
		}
	}

	// Try to extract from any field that might contain date information
	log.Printf("Checking all EXIF fields for date information in %s", filepath.Base(filePath))
	
	// Create a walker to find date fields
	walker := &dateWalker{}
	if err := exifData.Walk(walker); err != nil {
		log.Printf("Error walking EXIF data for %s: %v", filepath.Base(filePath), err)
	}
	
	if !walker.foundDate.IsZero() {
		return walker.foundDate
	}

	// No date fields found - this is normal for many images
	return time.Time{} // Return zero time if no date fields found
}


func (s *GalleryService) extractPhotoTimeWithExifTool(filePath string) time.Time {
	// Check if exiftool is available
	if _, err := exec.LookPath("exiftool"); err != nil {
		// exiftool not available, skip this method
		return time.Time{}
	}

	// Try to extract date fields using exiftool
	dateFields := []string{
		"DateTimeOriginal",
		"CreateDate",
		"DateTimeCreated",
		"MetadataDate",
		"DateTime",
		"DateTimeDigitized",
		"ModifyDate",
	}

	for _, field := range dateFields {
		cmd := exec.Command("exiftool", "-s", "-s", "-s", "-"+field, filePath)
		output, err := cmd.Output()
		if err != nil {
			continue // Field not found or error, try next field
		}

		dateStr := strings.TrimSpace(string(output))
		if dateStr == "" || dateStr == "-" {
			continue // Empty or no value
		}

		// Try to parse the date string with various formats
		dateFormats := []string{
			"2006:01:02 15:04:05-07:00", // With timezone
			"2006:01:02 15:04:05+07:00", // With timezone
			"2006:01:02 15:04:05",       // Standard EXIF format
			"2006-01-02 15:04:05",       // Alternative format
			"2006:01:02T15:04:05-07:00", // ISO-like with timezone
			"2006-01-02T15:04:05-07:00", // ISO with timezone
			"2006:01:02T15:04:05",       // ISO-like
			"2006-01-02T15:04:05",       // ISO format
			"2006:01:02",                // Date only
			"2006-01-02",                // Date only alternative
		}

		for _, format := range dateFormats {
			if photoTime, err := time.Parse(format, dateStr); err == nil {
				log.Printf("Extracted photo time from exiftool field %s for %s: %s", field, filepath.Base(filePath), photoTime.Format(time.RFC3339))
				return photoTime
			}
		}

		log.Printf("Found exiftool field %s for %s but couldn't parse date: %s", field, filepath.Base(filePath), dateStr)
	}

	return time.Time{} // No date found
}
