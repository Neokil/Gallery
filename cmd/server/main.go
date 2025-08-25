package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"photo-gallery/internal/api"
	"photo-gallery/internal/handlers"
	"photo-gallery/internal/service"
)

func main() {
	// Environment variables
	siteTitle := getEnv("SITE_TITLE", "Photo Gallery")
	password := getEnv("GALLERY_PASSWORD", "")
	uploadDir := getEnv("UPLOAD_DIR", "./uploads")
	metadataDir := getEnv("METADATA_DIR", "./metadata")
	port := getEnv("PORT", "8080")

	if password == "" {
		log.Fatal("GALLERY_PASSWORD environment variable is required")
	}

	// Create directories
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		log.Fatal("Failed to create metadata directory:", err)
	}

	// Initialize services
	galleryService := service.NewGalleryService(uploadDir, metadataDir)
	authService := service.NewAuthService(password)

	// Clean up orphaned metadata files
	galleryService.CleanupOrphanedMetadata()

	// Initialize handlers
	h, err := handlers.NewHandlers(galleryService, authService, siteTitle)
	if err != nil {
		log.Fatal("Failed to initialize handlers:", err)
	}

	// Create Chi router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(addSecurityHeaders)

	// Create regular server wrapper for session handling
	serverWrapper := &ServerWrapper{
		handlers: h,
	}

	// Mount the API routes
	api.HandlerFromMux(serverWrapper, r)

	log.Printf("Server starting on port %s", port)
	log.Printf("Site title: %s", siteTitle)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// ServerWrapper implements the regular ServerInterface for session handling
type ServerWrapper struct {
	handlers *handlers.Handlers
}

func (s *ServerWrapper) GetGallery(w http.ResponseWriter, r *http.Request, params api.GetGalleryParams) {
	s.handlers.HandleGallery(w, r, params)
}

func (s *ServerWrapper) GetLogin(w http.ResponseWriter, r *http.Request) {
	s.handlers.HandleGetLogin(w, r)
}

func (s *ServerWrapper) PostLogin(w http.ResponseWriter, r *http.Request) {
	s.handlers.HandlePostLogin(w, r)
}

func (s *ServerWrapper) UploadPhotos(w http.ResponseWriter, r *http.Request) {
	s.handlers.HandleUpload(w, r)
}

func (s *ServerWrapper) DownloadAllPhotos(w http.ResponseWriter, r *http.Request, params api.DownloadAllPhotosParams) {
	s.handlers.HandleDownloadAll(w, r, params)
}

func (s *ServerWrapper) ServePhoto(w http.ResponseWriter, r *http.Request, filename string) {
	s.handlers.HandleServePhoto(w, r, filename)
}

func (s *ServerWrapper) ServeStatic(w http.ResponseWriter, r *http.Request, filename string) {
	s.handlers.HandleServeStatic(w, r, filename)
}
