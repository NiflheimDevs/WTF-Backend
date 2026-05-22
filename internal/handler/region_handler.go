package handler

import (
	"net/http"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type RegionHandler struct {
	regions *service.RegionService
}

func NewRegionHandler(regions *service.RegionService) *RegionHandler {
	return &RegionHandler{regions: regions}
}

func (h *RegionHandler) List(w http.ResponseWriter, r *http.Request) {
	regions, err := h.regions.ListActive(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "regions_failed", "failed to load regions", nil)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	WriteJSON(w, http.StatusOK, regions)
}
