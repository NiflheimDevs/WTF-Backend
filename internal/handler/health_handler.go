package handler

import (
	"context"
	"net/http"
	"time"
)

type HealthChecker interface {
	Health(ctx context.Context) error
}

type HealthHandler struct {
	db HealthChecker
}

func NewHealthHandler(db HealthChecker) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Health(ctx); err != nil {
		WriteError(w, http.StatusServiceUnavailable, "not_ready", "database is not reachable", nil)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
