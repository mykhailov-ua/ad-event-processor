package fraudadmin

import (
	"errors"
	"net/http"
	"strings"

	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) registerCrowdWaveRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.CrowdWaves == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/crowd-waves/{campaign_id}", limit(perm("audit:read", h.getCrowdWaveSummary)))
}

func (h *HTTPHandlers) getCrowdWaveSummary(w http.ResponseWriter, r *http.Request) {
	campaignID := strings.TrimSpace(r.PathValue("campaign_id"))
	dto, err := h.CrowdWaves.GetSummary(r.Context(), campaignID)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}
