package handler

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/domain"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/middleware"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/repository"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type RequestHandler struct {
	requests *service.RequestService
}

func NewRequestHandler(requests *service.RequestService) *RequestHandler {
	return &RequestHandler{requests: requests}
}

type createRequestBody struct {
	RegionID     string `json:"region_id"`
	NeedType     string `json:"need_type"`
	Quantity     int    `json:"quantity"`
	ContactPhone string `json:"contact_phone"`
	Note         string `json:"note"`
}

type updateStatusBody struct {
	Status string `json:"status"`
}

func (h *RequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createRequestBody
	if err := DecodeJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}

	regionID, err := uuid.Parse(body.RegionID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "validation_failed", "region_id must be a valid UUID", nil)
		return
	}

	needType := domain.NeedType(body.NeedType)
	if !validNeedType(needType) {
		WriteError(w, http.StatusBadRequest, "validation_failed", "need_type must be bottled_water or tanker", nil)
		return
	}

	if body.Quantity < 1 {
		WriteError(w, http.StatusBadRequest, "validation_failed", "quantity must be greater than zero", nil)
		return
	}

	if len(body.Note) > 500 {
		WriteError(w, http.StatusBadRequest, "validation_failed", "note must be at most 500 characters", nil)
		return
	}

	contactPhone := optionalString(body.ContactPhone)
	note := optionalString(body.Note)
	userAgent := optionalString(r.UserAgent())
	ip := optionalString(clientIP(r))

	req, err := h.requests.Create(r.Context(), service.CreateRequestInput{
		RegionID:           regionID,
		NeedType:           needType,
		Quantity:           body.Quantity,
		ContactPhone:       contactPhone,
		Note:               note,
		SubmittedIP:        ip,
		SubmittedUserAgent: userAgent,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			WriteError(w, http.StatusBadRequest, "invalid_region", "region_id does not reference an active region", nil)
			return
		}
		WriteError(w, http.StatusInternalServerError, "request_create_failed", "failed to create request", nil)
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{
		"id":         req.ID,
		"status":     req.Status,
		"created_at": req.CreatedAt,
	})
}

func (h *RequestHandler) List(w http.ResponseWriter, r *http.Request) {
	filters, page, pageSize, ok := parseRequestFilters(w, r)
	if !ok {
		return
	}

	result, err := h.requests.List(r.Context(), filters, page, pageSize)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "requests_failed", "failed to load requests", nil)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func (h *RequestHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.requests.Detail(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "not_found", "request not found", nil)
			return
		}
		WriteError(w, http.StatusInternalServerError, "request_failed", "failed to load request", nil)
		return
	}

	WriteJSON(w, http.StatusOK, detail)
}

func (h *RequestHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication is required", nil)
		return
	}

	actorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token subject", nil)
		return
	}

	var body updateStatusBody
	if err := DecodeJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}

	status := domain.RequestStatus(body.Status)
	if !validDispatcherStatus(status) {
		WriteError(w, http.StatusBadRequest, "validation_failed", "status must be dispatched, fulfilled, or cancelled", nil)
		return
	}

	updated, err := h.requests.UpdateStatus(r.Context(), id, status, actorID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			WriteError(w, http.StatusNotFound, "not_found", "request not found", nil)
		case errors.Is(err, service.ErrInvalidTransition):
			WriteError(w, http.StatusConflict, "invalid_transition", "requested status transition is not allowed", nil)
		default:
			WriteError(w, http.StatusInternalServerError, "status_update_failed", "failed to update request status", nil)
		}
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

func parseRequestFilters(w http.ResponseWriter, r *http.Request) (repository.RequestFilters, int, int, bool) {
	query := r.URL.Query()
	filters := repository.RequestFilters{}

	if raw := query.Get("status"); raw != "" {
		status := domain.RequestStatus(raw)
		if !validRequestStatus(status) {
			WriteError(w, http.StatusBadRequest, "validation_failed", "status filter is invalid", nil)
			return filters, 0, 0, false
		}
		filters.Status = &status
	}

	if raw := query.Get("region_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "validation_failed", "region_id filter must be a UUID", nil)
			return filters, 0, 0, false
		}
		filters.RegionID = &id
	}

	if raw := query.Get("from"); raw != "" {
		from, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "validation_failed", "from must be an RFC3339 timestamp", nil)
			return filters, 0, 0, false
		}
		filters.FromDate = &from
	}

	if raw := query.Get("to"); raw != "" {
		to, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "validation_failed", "to must be an RFC3339 timestamp", nil)
			return filters, 0, 0, false
		}
		filters.ToDate = &to
	}

	page := parsePositiveInt(query.Get("page"), 1)
	pageSize := parsePositiveInt(query.Get("page_size"), 20)
	return filters, page, pageSize, true
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "validation_failed", name+" must be a valid UUID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func validNeedType(value domain.NeedType) bool {
	return value == domain.NeedTypeBottledWater || value == domain.NeedTypeTanker
}

func validDispatcherStatus(value domain.RequestStatus) bool {
	return value == domain.StatusDispatched || value == domain.StatusFulfilled || value == domain.StatusCancelled
}

func validRequestStatus(value domain.RequestStatus) bool {
	return value == domain.StatusPending || value == domain.StatusDispatched || value == domain.StatusFulfilled || value == domain.StatusCancelled
}
