package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	filePermissions = 0600 // File permissions for metadata files
)

type PhotoInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Uploader string    `json:"uploader"`
	Event    string    `json:"event"`
	Date     time.Time `json:"date"`
}

type GalleryService struct {
	uploadDir   string
	metadataDir string
}

func NewGalleryService(uploadDir, metadataDir string) *GalleryService {
	service := &GalleryService{
		uploadDir:   uploadDir,
		metadataDir: metadataDir,
	}

	// Generate metadata for existing images on startup
	service.GenerateMissingMetadata()

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
				// Fallback for photos without metadata
				photoInfo = PhotoInfo{
					Path:     "/uploads/" + file.Name(),
					Name:     file.Name(),
					Uploader: "Unknown",
					Event:    "",
					Date:     time.Now(),
				}
			}
			photos = append(photos, photoInfo)
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

	// Save photo metadata
	photoInfo := PhotoInfo{
		Path:     "/uploads/" + filename,
		Name:     filename,
		Uploader: userName,
		Event:    eventName,
		Date:     time.Now(),
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

		// Generate default metadata
		photoInfo := PhotoInfo{
			Path:     "/uploads/" + file.Name(),
			Name:     file.Name(),
			Uploader: "Unknown",
			Event:    "",
			Date:     fileInfo.ModTime(),
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
