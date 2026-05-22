package middleware

import (
	"net/http"
	"strings"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type TokenValidator interface {
	ValidateToken(tokenString string) (*service.Claims, error)
}

func Auth(auth TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized", "bearer token is required")
				return
			}

			claims, err := auth.ValidateToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "token is invalid or expired")
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
		})
	}
}
