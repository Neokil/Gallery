package service

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewGalleryService(t *testing.T) {
	uploadDir := "test_uploads"
	metadataDir := "test_metadata"

	// Clean up test directories
	defer func() {
		os.RemoveAll(uploadDir)
		os.RemoveAll(metadataDir)
	}()

	service := NewGalleryService(uploadDir, metadataDir)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.uploadDir != uploadDir {
		t.Errorf("Expected uploadDir to be %s, got %s", uploadDir, service.uploadDir)
	}

	if service.metadataDir != metadataDir {
		t.Errorf("Expected metadataDir to be %s, got %s", metadataDir, service.metadataDir)
	}
}

func TestFilterPhotos(t *testing.T) {
	service := NewGalleryService("uploads", "metadata")

	photos := []PhotoInfo{
		{
			Name:     "photo1.jpg",
			Uploader: "Alice",
			Event:    "Birthday",
			Date:     time.Now(),
		},
		{
			Name:     "photo2.jpg",
			Uploader: "Bob",
			Event:    "Wedding",
			Date:     time.Now(),
		},
		{
			Name:     "photo3.jpg",
			Uploader: "Alice",
			Event:    "Birthday",
			Date:     time.Now(),
		},
	}

	// Test event filter
	filtered := service.FilterPhotos(photos, "Birthday", "")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 photos for Birthday event, got %d", len(filtered))
	}

	// Test uploader filter
	filtered = service.FilterPhotos(photos, "", "Alice")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 photos for Alice uploader, got %d", len(filtered))
	}

	// Test both filters
	filtered = service.FilterPhotos(photos, "Birthday", "Alice")
	if len(filtered) != 2 {
		t.Errorf("Expected 2 photos for Birthday event and Alice uploader, got %d", len(filtered))
	}

	// Test no filters
	filtered = service.FilterPhotos(photos, "", "")
	if len(filtered) != 3 {
		t.Errorf("Expected 3 photos with no filters, got %d", len(filtered))
	}
}

func TestGetUniqueEvents(t *testing.T) {
	service := NewGalleryService("uploads", "metadata")

	photos := []PhotoInfo{
		{Event: "Birthday"},
		{Event: "Wedding"},
		{Event: "Birthday"},
		{Event: "Party"},
	}

	events := service.GetUniqueEvents(photos)

	if len(events) != 3 {
		t.Errorf("Expected 3 unique events, got %d", len(events))
	}

	expectedEvents := map[string]bool{
		"Birthday": true,
		"Wedding":  true,
		"Party":    true,
	}

	for _, event := range events {
		if !expectedEvents[event] {
			t.Errorf("Unexpected event: %s", event)
		}
	}
}

func TestGetUniqueUploaders(t *testing.T) {
	service := NewGalleryService("uploads", "metadata")

	photos := []PhotoInfo{
		{Uploader: "Alice"},
		{Uploader: "Bob"},
		{Uploader: "Alice"},
		{Uploader: "Charlie"},
	}

	uploaders := service.GetUniqueUploaders(photos)

	if len(uploaders) != 3 {
		t.Errorf("Expected 3 unique uploaders, got %d", len(uploaders))
	}

	expectedUploaders := map[string]bool{
		"Alice":   true,
		"Bob":     true,
		"Charlie": true,
	}

	for _, uploader := range uploaders {
		if !expectedUploaders[uploader] {
			t.Errorf("Unexpected uploader: %s", uploader)
		}
	}
}

func TestServePhoto(t *testing.T) {
	uploadDir := "test_uploads"
	service := NewGalleryService(uploadDir, "metadata")

	// Clean up test directory
	defer os.RemoveAll(uploadDir)

	// Create test directory and file
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(uploadDir, "test.jpg")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test existing file
	path, err := service.ServePhoto("test.jpg")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, path)
	}

	// Test non-existing file
	_, err = service.ServePhoto("nonexistent.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGenerateMissingMetadata(t *testing.T) {
	uploadDir := "test_uploads_metadata"
	metadataDir := "test_metadata_metadata"

	// Clean up test directories
	defer func() {
		os.RemoveAll(uploadDir)
		os.RemoveAll(metadataDir)
	}()

	// Create test directories
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create test image files without metadata
	testFiles := []string{"test1.png", "test2.png", "test3.png"}
	for _, filename := range testFiles {
		testFile := filepath.Join(uploadDir, filename)
		err = createTestPNG(testFile)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create service (this should trigger metadata generation)
	service := NewGalleryService(uploadDir, metadataDir)

	// Verify metadata files were created
	for _, filename := range testFiles {
		metadataFile := filepath.Join(metadataDir, filename+".json")
		if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
			t.Errorf("Expected metadata file %s to be created", metadataFile)
		}

		// Verify metadata content
		photoInfo := service.loadPhotoMetadata(filename)
		if photoInfo.Name != filename {
			t.Errorf("Expected photo name %s, got %s", filename, photoInfo.Name)
		}
		if photoInfo.Uploader != "Unknown" {
			t.Errorf("Expected uploader to be 'Unknown', got %s", photoInfo.Uploader)
		}
		if photoInfo.Path != "/uploads/"+filename {
			t.Errorf("Expected path to be '/uploads/%s', got %s", filename, photoInfo.Path)
		}
	}
}

func TestGenerateMissingMetadataSkipsExisting(t *testing.T) {
	uploadDir := "test_uploads_existing"
	metadataDir := "test_metadata_existing"

	// Clean up test directories
	defer func() {
		os.RemoveAll(uploadDir)
		os.RemoveAll(metadataDir)
	}()

	// Create test directories
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(metadataDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create test image file
	testFile := filepath.Join(uploadDir, "existing.png")
	err = createTestPNG(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Create existing metadata with custom values
	existingMetadata := PhotoInfo{
		Path:     "/uploads/existing.png",
		Name:     "existing.png",
		Uploader: "TestUser",
		Event:    "TestEvent",
		Date:     time.Now(),
	}

	service := &GalleryService{
		uploadDir:   uploadDir,
		metadataDir: metadataDir,
	}
	service.savePhotoMetadata("existing.png", &existingMetadata)

	// Now create service (should not overwrite existing metadata)
	service = NewGalleryService(uploadDir, metadataDir)

	// Verify existing metadata was preserved
	photoInfo := service.loadPhotoMetadata("existing.png")
	if photoInfo.Uploader != "TestUser" {
		t.Errorf("Expected uploader to remain 'TestUser', got %s", photoInfo.Uploader)
	}
	if photoInfo.Event != "TestEvent" {
		t.Errorf("Expected event to remain 'TestEvent', got %s", photoInfo.Event)
	}
}

func TestServeThumbnail(t *testing.T) {
	uploadDir := "test_uploads_thumb"
	metadataDir := "test_metadata_thumb"
	thumbnailDir := filepath.Join(metadataDir, "thumbnails")

	// Clean up test directories
	defer func() {
		os.RemoveAll(uploadDir)
		os.RemoveAll(metadataDir)
	}()

	// Create test directories
	err := os.MkdirAll(thumbnailDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create test thumbnail file
	testThumbnail := filepath.Join(thumbnailDir, "test.jpg")
	err = os.WriteFile(testThumbnail, []byte("test thumbnail content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	service := &GalleryService{
		uploadDir:    uploadDir,
		metadataDir:  metadataDir,
		thumbnailDir: thumbnailDir,
	}

	// Test existing thumbnail
	path, err := service.ServeThumbnail("test.jpg")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if path != testThumbnail {
		t.Errorf("Expected path %s, got %s", testThumbnail, path)
	}

	// Test non-existing thumbnail
	_, err = service.ServeThumbnail("nonexistent.jpg")
	if err == nil {
		t.Error("Expected error for non-existent thumbnail, got nil")
	}
}

func TestGenerateThumbnail(t *testing.T) {
	uploadDir := "test_uploads_gen_thumb"
	metadataDir := "test_metadata_gen_thumb"
	thumbnailDir := filepath.Join(metadataDir, "thumbnails")

	// Clean up test directories
	defer func() {
		os.RemoveAll(uploadDir)
		os.RemoveAll(metadataDir)
	}()

	// Create test directories
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(thumbnailDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	service := &GalleryService{
		uploadDir:    uploadDir,
		metadataDir:  metadataDir,
		thumbnailDir: thumbnailDir,
	}

	// Create a proper test PNG image programmatically
	testImagePath := filepath.Join(uploadDir, "test.png")
	err = createTestPNG(testImagePath)
	if err != nil {
		t.Fatal(err)
	}

	// Generate thumbnail
	thumbnailPath := filepath.Join(thumbnailDir, "test.png")
	err = service.generateThumbnail(testImagePath, thumbnailPath)
	if err != nil {
		t.Errorf("Expected no error generating thumbnail, got %v", err)
	}

	// Verify thumbnail was created
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		t.Error("Expected thumbnail file to be created")
	}
}

// Helper function to create a valid test PNG image
func createTestPNG(filename string) error {
	// Create a 10x10 red image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, red) // Red pixel
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func TestCleanupOrphanedThumbnails(t *testing.T) {
	uploadDir := "test_uploads_cleanup_thumb"
	metadataDir := "test_metadata_cleanup_thumb"
	thumbnailDir := filepath.Join(metadataDir, "thumbnails")

	// Clean up test directories
	defer func() {
		os.RemoveAll(uploadDir)
		os.RemoveAll(metadataDir)
	}()

	// Create test directories
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(thumbnailDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	service := &GalleryService{
		uploadDir:    uploadDir,
		metadataDir:  metadataDir,
		thumbnailDir: thumbnailDir,
	}

	// Create a valid image and its thumbnail
	validImagePath := filepath.Join(uploadDir, "valid.png")
	err = createTestPNG(validImagePath)
	if err != nil {
		t.Fatal(err)
	}

	validThumbnailPath := filepath.Join(thumbnailDir, "valid.png")
	err = createTestPNG(validThumbnailPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create an orphaned thumbnail (no corresponding original image)
	orphanedThumbnailPath := filepath.Join(thumbnailDir, "orphaned.png")
	err = createTestPNG(orphanedThumbnailPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify both thumbnails exist before cleanup
	if _, err := os.Stat(validThumbnailPath); os.IsNotExist(err) {
		t.Fatal("Valid thumbnail should exist before cleanup")
	}
	if _, err := os.Stat(orphanedThumbnailPath); os.IsNotExist(err) {
		t.Fatal("Orphaned thumbnail should exist before cleanup")
	}

	// Run cleanup
	service.CleanupOrphanedThumbnails()

	// Verify valid thumbnail still exists
	if _, err := os.Stat(validThumbnailPath); os.IsNotExist(err) {
		t.Error("Valid thumbnail should still exist after cleanup")
	}

	// Verify orphaned thumbnail was removed
	if _, err := os.Stat(orphanedThumbnailPath); err == nil {
		t.Error("Orphaned thumbnail should have been removed")
	}
}
