package main

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
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

var (
	store     = sessions.NewCookieStore([]byte(generateSecretKey()))
	templates *template.Template
	siteTitle = getEnv("SITE_TITLE", "Photo Gallery")
	password  = getEnv("GALLERY_PASSWORD", "")
	uploadDir = getEnv("UPLOAD_DIR", "./uploads")
)

func main() {
	if password == "" {
		log.Fatal("GALLERY_PASSWORD environment variable is required")
	}

	// Create upload directory
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
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
	photos, err := getPhotos()
	if err != nil {
		http.Error(w, "Failed to load photos", http.StatusInternalServerError)
		return
	}

	templates.ExecuteTemplate(w, "gallery.html", map[string]interface{}{
		"Title":  siteTitle,
		"Photos": photos,
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(32 << 20) // 32MB max

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

		// Generate unique filename
		ext := filepath.Ext(fileHeader.Filename)
		filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), generateRandomString(8), ext)
		filepath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filepath)
		if err != nil {
			continue
		}
		defer dst.Close()

		io.Copy(dst, file)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func serveUploads(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}

func getPhotos() ([]string, error) {
	var photos []string

	files, err := os.ReadDir(uploadDir)
	if err != nil {
		return photos, err
	}

	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			photos = append(photos, "/uploads/"+file.Name())
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
