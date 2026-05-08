package handler

import (
	"net/http"
	"strconv"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type MetricsHandler struct {
	metrics *service.MetricsService
}

func NewMetricsHandler(metrics *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{metrics: metrics}
}

func (h *MetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.metrics.Summary(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "metrics_failed", "failed to load summary metrics", nil)
		return
	}
	WriteJSON(w, http.StatusOK, summary)
}

func (h *MetricsHandler) ByRegion(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	metrics, err := h.metrics.ByRegion(r.Context(), limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "metrics_failed", "failed to load region metrics", nil)
		return
	}
	WriteJSON(w, http.StatusOK, metrics)
}

func (h *MetricsHandler) ByNeedType(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metrics.ByNeedType(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "metrics_failed", "failed to load need type metrics", nil)
		return
	}
	WriteJSON(w, http.StatusOK, metrics)
}
