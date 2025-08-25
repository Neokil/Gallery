// Package service provides business logic services for the photo gallery application.
package service

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	secretKeyLength = 32 // Length of secret key in bytes
)

type AuthService struct {
	store    *sessions.CookieStore
	Password string
}

func NewAuthService(password, sessionKey string) *AuthService {
	// Use provided session key or generate one if empty
	key := sessionKey
	if key == "" {
		key = generateSecretKey()
	}

	store := sessions.NewCookieStore([]byte(key))
	// Configure session options for better compatibility
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Will be set dynamically based on request
		SameSite: http.SameSiteLaxMode,
		Domain:   "", // Empty domain works better with IP addresses
	}

	return &AuthService{
		store:    store,
		Password: password,
	}
}

func (a *AuthService) IsAuthenticated(r *http.Request) bool {
	session, err := a.store.Get(r, "gallery-session")
	if err != nil {
		// Log session retrieval errors for debugging
		return false
	}

	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		return true
	}
	return false
}

func (a *AuthService) Login(w http.ResponseWriter, r *http.Request, password string) bool {
	if password != a.Password {
		return false
	}

	session, err := a.store.Get(r, "gallery-session")
	if err != nil {
		return false
	}

	// Set secure cookie if using HTTPS
	session.Options.Secure = r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil

	session.Values["authenticated"] = true
	if err := session.Save(r, w); err != nil {
		return false
	}
	return true
}

func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := a.store.Get(r, "gallery-session")
	session.Values["authenticated"] = false

	// Set MaxAge to -1 to delete the cookie immediately
	session.Options.MaxAge = -1

	_ = session.Save(r, w) // Ignore error on logout
}

func generateSecretKey() string {
	key := make([]byte, secretKeyLength)
	if _, err := rand.Read(key); err != nil {
		// Fallback to a default key if random generation fails
		return "default-secret-key-change-in-production"
	}
	return base64.StdEncoding.EncodeToString(key)
}
