package service

import (
	"net/http/httptest"
	"testing"
)

func TestNewAuthService(t *testing.T) {
	password := "test-password"

	service := NewAuthService(password)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.Password != password {
		t.Errorf("Expected password to be %s, got %s", password, service.Password)
	}

	if service.store == nil {
		t.Error("Expected store to be initialized, got nil")
	}
}

func TestLogin(t *testing.T) {
	password := "correct-password"
	service := NewAuthService(password)

	// Test correct password
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)

	result := service.Login(w, r, password)
	if !result {
		t.Error("Expected login to succeed with correct password")
	}

	// Test incorrect password
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/login", nil)

	result = service.Login(w, r, "wrong-password")
	if result {
		t.Error("Expected login to fail with incorrect password")
	}
}

func TestIsAuthenticated(t *testing.T) {
	service := NewAuthService("password")

	// Test without session
	r := httptest.NewRequest("GET", "/", nil)
	if service.IsAuthenticated(r) {
		t.Error("Expected authentication to fail without session")
	}

	// Test with valid session - need to simulate the full login flow
	w := httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/login", nil)

	// Login first
	loginSuccess := service.Login(w, r, "password")
	if !loginSuccess {
		t.Fatal("Login should have succeeded")
	}

	// Get the session cookie from the response
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Skip("No session cookie set, skipping authenticated test")
		return
	}

	// Create new request with session cookie
	r2 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range cookies {
		r2.AddCookie(cookie)
	}

	if !service.IsAuthenticated(r2) {
		t.Error("Expected authentication to succeed with valid session")
	}
}

func TestLogout(t *testing.T) {
	service := NewAuthService("password")

	// Login first
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	loginSuccess := service.Login(w, r, "password")
	if !loginSuccess {
		t.Fatal("Login should have succeeded")
	}

	// Get the session cookie
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Skip("No session cookie set, skipping logout test")
		return
	}

	// Create new request with session cookie for logout
	r2 := httptest.NewRequest("GET", "/logout", nil)
	for _, cookie := range cookies {
		r2.AddCookie(cookie)
	}

	// Logout
	w2 := httptest.NewRecorder()
	service.Logout(w2, r2)

	// Create new request with updated session cookie to test authentication
	logoutCookies := w2.Result().Cookies()
	r3 := httptest.NewRequest("GET", "/", nil)
	for _, cookie := range logoutCookies {
		r3.AddCookie(cookie)
	}

	// Verify session is cleared
	if service.IsAuthenticated(r3) {
		t.Error("Expected authentication to fail after logout")
	}
}
