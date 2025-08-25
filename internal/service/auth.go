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

func NewAuthService(password string) *AuthService {
	return &AuthService{
		store:    sessions.NewCookieStore([]byte(generateSecretKey())),
		Password: password,
	}
}

func (a *AuthService) IsAuthenticated(r *http.Request) bool {
	session, _ := a.store.Get(r, "gallery-session")
	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		return true
	}
	return false
}

func (a *AuthService) Login(w http.ResponseWriter, r *http.Request, password string) bool {
	if password != a.Password {
		return false
	}

	session, _ := a.store.Get(r, "gallery-session")
	session.Values["authenticated"] = true
	if err := session.Save(r, w); err != nil {
		return false
	}
	return true
}

func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := a.store.Get(r, "gallery-session")
	session.Values["authenticated"] = false
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
