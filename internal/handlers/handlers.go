// Package handlers provides HTTP request handlers for the photo gallery application.
package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Neokil/Gallery/internal/api"
	"github.com/Neokil/Gallery/internal/service"
)

const (
	maxUploadSize = 32 << 20 // 32MB max upload size
)

type Handlers struct {
	galleryService *service.GalleryService
	authService    *service.AuthService
	templates      *template.Template
	siteTitle      string
}

func NewHandlers(galleryService *service.GalleryService, authService *service.AuthService, siteTitle string) (*Handlers, error) {
	templates, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Handlers{
		galleryService: galleryService,
		authService:    authService,
		templates:      templates,
		siteTitle:      siteTitle,
	}, nil
}

// HandleGallery implements the gallery page handler
func (h *Handlers) HandleGallery(w http.ResponseWriter, r *http.Request, params api.GetGalleryParams) {
	// Check authentication
	if !h.authService.IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get filter parameters
	var eventFilter, uploaderFilter string
	if params.Event != nil {
		eventFilter = *params.Event
	}
	if params.Uploader != nil {
		uploaderFilter = *params.Uploader
	}

	photos, err := h.galleryService.GetPhotos()
	if err != nil {
		http.Error(w, "Failed to load photos", http.StatusInternalServerError)
		return
	}

	// Apply filters
	filteredPhotos := h.galleryService.FilterPhotos(photos, eventFilter, uploaderFilter)

	// Get unique events and uploaders for filter dropdowns
	events := h.galleryService.GetUniqueEvents(photos)
	uploaders := h.galleryService.GetUniqueUploaders(photos)

	// Render template
	data := map[string]any{
		"Title":            h.siteTitle,
		"Photos":           filteredPhotos,
		"AllEvents":        events,
		"AllUploaders":     uploaders,
		"SelectedEvent":    eventFilter,
		"SelectedUploader": uploaderFilter,
		"TotalPhotos":      len(photos),
		"FilteredPhotos":   len(filteredPhotos),
		"CacheBreaker":     time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "gallery.html", data); err != nil {
		log.Printf("Failed to execute gallery template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleGetLogin implements the login page handler
func (h *Handlers) HandleGetLogin(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":        h.siteTitle,
		"CacheBreaker": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "login.html", data); err != nil {
		log.Printf("Failed to execute login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandlePostLogin implements the login form submission handler
func (h *Handlers) HandlePostLogin(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")

	if h.authService.Login(w, r, password) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Login failed
	data := map[string]any{
		"Title":        h.siteTitle,
		"Error":        "Invalid password",
		"CacheBreaker": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "login.html", data); err != nil {
		log.Printf("Failed to execute login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandleUpload implements the photo upload handler
func (h *Handlers) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !h.authService.IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Get uploader name and event name from form
	userName := strings.TrimSpace(r.FormValue("uploader_name"))
	if userName == "" {
		userName = "Anonymous"
	}
	eventName := strings.TrimSpace(r.FormValue("event_name"))

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range files {
		err := h.galleryService.SavePhoto(fileHeader, userName, eventName)
		if err != nil {
			log.Printf("Failed to save photo %s: %v", fileHeader.Filename, err)
			continue
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleDownloadAll implements the download all photos handler
func (h *Handlers) HandleDownloadAll(w http.ResponseWriter, r *http.Request, params api.DownloadAllPhotosParams) {
	// Check authentication
	if !h.authService.IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filter parameters
	var eventFilter, uploaderFilter string
	if params.Event != nil {
		eventFilter = *params.Event
	}
	if params.Uploader != nil {
		uploaderFilter = *params.Uploader
	}

	// Get all photos and apply filters
	photos, err := h.galleryService.GetPhotos()
	if err != nil {
		http.Error(w, "Failed to load photos", http.StatusInternalServerError)
		return
	}

	filteredPhotos := h.galleryService.FilterPhotos(photos, eventFilter, uploaderFilter)

	if len(filteredPhotos) == 0 {
		http.Error(w, "No photos to download", http.StatusNotFound)
		return
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	var filename string
	if eventFilter != "" || uploaderFilter != "" {
		filterSuffix := ""
		if eventFilter != "" {
			filterSuffix += "_" + strings.ReplaceAll(eventFilter, " ", "_")
		}
		if uploaderFilter != "" {
			filterSuffix += "_" + strings.ReplaceAll(uploaderFilter, " ", "_")
		}
		filename = fmt.Sprintf("gallery_photos%s_%s.zip", filterSuffix, timestamp)
	} else {
		filename = fmt.Sprintf("gallery_photos_%s.zip", timestamp)
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if err := h.galleryService.CreateZipArchive(filteredPhotos, w); err != nil {
		log.Printf("Failed to create zip archive: %v", err)
		http.Error(w, "Failed to create archive", http.StatusInternalServerError)
	}
}

// HandleServePhoto implements the photo serving handler
func (h *Handlers) HandleServePhoto(w http.ResponseWriter, r *http.Request, filename string) {
	// Check authentication before serving photos
	if !h.authService.IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filePath, err := h.galleryService.ServePhoto(filename)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

// HandleServeStatic implements the static file serving handler
func (h *Handlers) HandleServeStatic(w http.ResponseWriter, r *http.Request, filename string) {
	filePath := filepath.Join("static", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}
