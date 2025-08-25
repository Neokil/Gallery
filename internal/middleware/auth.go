// Package middleware provides HTTP middleware for the photo gallery application.
package middleware

import (
	"net/http"

	"photo-gallery/internal/service"
)

func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for login, logout, and static files
			if r.URL.Path == "/login" || r.URL.Path == "/logout" ||
				r.URL.Path == "/static/" || r.URL.Path == "/uploads/" {
				next.ServeHTTP(w, r)
				return
			}

			if !authService.IsAuthenticated(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
