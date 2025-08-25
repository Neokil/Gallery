package service

import (
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
