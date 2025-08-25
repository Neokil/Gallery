package service

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gorilla/sessions"
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
	session.Save(r, w)
	return true
}

func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := a.store.Get(r, "gallery-session")
	session.Values["authenticated"] = false
	session.Save(r, w)
}

func generateSecretKey() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}
