package main

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/sessions"
)

type PhotoInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Uploader string    `json:"uploader"`
	Event    string    `json:"event"`
	Date     time.Time `json:"date"`
}

var (
	store       = sessions.NewCookieStore([]byte(generateSecretKey()))
	templates   *template.Template
	siteTitle   = getEnv("SITE_TITLE", "Photo Gallery")
	password    = getEnv("GALLERY_PASSWORD", "")
	uploadDir   = getEnv("UPLOAD_DIR", "./uploads")
	metadataDir = getEnv("METADATA_DIR", "./metadata")
)

func main() {
	if password == "" {
		log.Fatal("GALLERY_PASSWORD environment variable is required")
	}

	// Create upload and metadata directories
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		log.Fatal("Failed to create metadata directory:", err)
	}

	// Parse templates
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Failed to parse templates:", err)
	}

	// Routes
	http.HandleFunc("/", authMiddleware(galleryHandler))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/upload", authMiddleware(uploadHandler))
	http.HandleFunc("/download-all", authMiddleware(downloadAllHandler))
	http.HandleFunc("/uploads/", serveUploads)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Printf("Site title: %s", siteTitle)
	log.Fatal(http.ListenAndServe(":"+port, addSecurityHeaders(http.DefaultServeMux)))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateSecretKey() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
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

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "gallery-session")
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "login.html", map[string]string{
			"Title": siteTitle,
		})
		return
	}

	if r.Method == "POST" {
		if r.FormValue("password") == password {
			session, _ := store.Get(r, "gallery-session")
			session.Values["authenticated"] = true

			// Store the user's name if provided
			name := strings.TrimSpace(r.FormValue("name"))
			if name != "" {
				session.Values["user_name"] = name
			}

			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Title": siteTitle,
			"Error": "Invalid password",
		})
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "gallery-session")
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters from query string
	eventFilter := r.URL.Query().Get("event")
	uploaderFilter := r.URL.Query().Get("uploader")

	photos, err := getPhotos()
	if err != nil {
		http.Error(w, "Failed to load photos", http.StatusInternalServerError)
		return
	}

	// Apply filters
	filteredPhotos := filterPhotos(photos, eventFilter, uploaderFilter)

	// Get unique events and uploaders for filter dropdowns
	events := getUniqueEvents(photos)
	uploaders := getUniqueUploaders(photos)

	templates.ExecuteTemplate(w, "gallery.html", map[string]interface{}{
		"Title":            siteTitle,
		"Photos":           filteredPhotos,
		"AllEvents":        events,
		"AllUploaders":     uploaders,
		"SelectedEvent":    eventFilter,
		"SelectedUploader": uploaderFilter,
		"TotalPhotos":      len(photos),
		"FilteredPhotos":   len(filteredPhotos),
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user name from session
	session, _ := store.Get(r, "gallery-session")
	userName := "Anonymous"
	if name, ok := session.Values["user_name"].(string); ok && name != "" {
		userName = name
	}

	r.ParseMultipartForm(32 << 20) // 32MB max

	// Get event name from form
	eventName := strings.TrimSpace(r.FormValue("event_name"))

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range files {
		if !isValidImageType(fileHeader.Header.Get("Content-Type")) {
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		// Generate unique filename preserving original name
		filename := generateUniqueFilename(fileHeader.Filename)
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			continue
		}
		defer dst.Close()

		io.Copy(dst, file)

		// Save photo metadata
		photoInfo := PhotoInfo{
			Path:     "/uploads/" + filename,
			Name:     filename,
			Uploader: userName,
			Event:    eventName,
			Date:     time.Now(),
		}
		savePhotoMetadata(filename, photoInfo)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func serveUploads(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}

func getPhotos() ([]PhotoInfo, error) {
	var photos []PhotoInfo

	files, err := os.ReadDir(uploadDir)
	if err != nil {
		return photos, err
	}

	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			photoInfo := loadPhotoMetadata(file.Name())
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

func isValidImageType(contentType string) bool {
	validTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"}
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

func generateUniqueFilename(originalFilename string) string {
	// Clean the filename to remove any path components
	originalFilename = filepath.Base(originalFilename)

	// Check if the original filename already exists
	filePath := filepath.Join(uploadDir, originalFilename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, use original name
		return originalFilename
	}

	// File exists, generate a unique name with suffix
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)

	counter := 1
	for {
		newFilename := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		filePath := filepath.Join(uploadDir, newFilename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return newFilename
		}
		counter++
	}
}

func savePhotoMetadata(filename string, info PhotoInfo) {
	metadataFile := filepath.Join(metadataDir, filename+".json")
	data, err := json.Marshal(info)
	if err != nil {
		log.Printf("Failed to marshal metadata for %s: %v", filename, err)
		return
	}

	err = os.WriteFile(metadataFile, data, 0644)
	if err != nil {
		log.Printf("Failed to save metadata for %s: %v", filename, err)
	}
}

func loadPhotoMetadata(filename string) PhotoInfo {
	metadataFile := filepath.Join(metadataDir, filename+".json")
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

func filterPhotos(photos []PhotoInfo, eventFilter, uploaderFilter string) []PhotoInfo {
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

func getUniqueEvents(photos []PhotoInfo) []string {
	eventSet := make(map[string]bool)
	var events []string

	for _, photo := range photos {
		if photo.Event != "" && !eventSet[photo.Event] {
			eventSet[photo.Event] = true
			events = append(events, photo.Event)
		}
	}

	// Sort events alphabetically
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[i] > events[j] {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	return events
}

func getUniqueUploaders(photos []PhotoInfo) []string {
	uploaderSet := make(map[string]bool)
	var uploaders []string

	for _, photo := range photos {
		if photo.Uploader != "" && !uploaderSet[photo.Uploader] {
			uploaderSet[photo.Uploader] = true
			uploaders = append(uploaders, photo.Uploader)
		}
	}

	// Sort uploaders alphabetically
	for i := 0; i < len(uploaders)-1; i++ {
		for j := i + 1; j < len(uploaders); j++ {
			if uploaders[i] > uploaders[j] {
				uploaders[i], uploaders[j] = uploaders[j], uploaders[i]
			}
		}
	}

	return uploaders
}

func downloadAllHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "Failed to read upload directory", http.StatusInternalServerError)
		return
	}

	// Filter for image files
	var imageFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			imageFiles = append(imageFiles, file)
		}
	}

	if len(imageFiles) == 0 {
		http.Error(w, "No photos to download", http.StatusNotFound)
		return
	}

	// Set headers for zip download
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("gallery_photos_%s.zip", timestamp)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Create zip writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Add each image to the zip
	for _, file := range imageFiles {
		filePath := filepath.Join(uploadDir, file.Name())

		// Open the file
		fileReader, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open file %s: %v", file.Name(), err)
			continue
		}

		// Create a file in the zip
		zipFile, err := zipWriter.Create(file.Name())
		if err != nil {
			log.Printf("Failed to create zip entry for %s: %v", file.Name(), err)
			fileReader.Close()
			continue
		}

		// Copy file content to zip
		_, err = io.Copy(zipFile, fileReader)
		if err != nil {
			log.Printf("Failed to copy file %s to zip: %v", file.Name(), err)
		}

		fileReader.Close()
	}
}
