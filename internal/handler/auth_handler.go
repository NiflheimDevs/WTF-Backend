package handler

import (
	"errors"
	"net/http"
	"strings"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if err := DecodeJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" || input.Password == "" {
		WriteError(w, http.StatusBadRequest, "validation_failed", "email and password are required", nil)
		return
	}

	result, err := h.auth.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		status := http.StatusUnauthorized
		code := "invalid_credentials"
		message := "email or password is incorrect"
		if errors.Is(err, service.ErrInactiveUser) {
			status = http.StatusForbidden
			code = "inactive_user"
			message = "user account is inactive"
		}
		WriteError(w, status, code, message, nil)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}
