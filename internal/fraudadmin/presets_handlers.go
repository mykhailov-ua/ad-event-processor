package fraudadmin

import (
	"net/http"

	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) registerFraudPresetRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, permAny func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Presets == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/presets", limit(permAny([]string{
		"campaigns:read",
		"campaigns:read:masked",
		"audit:read",
		"shards:read",
	}, h.listFraudPresets)))
}

func (h *HTTPHandlers) listFraudPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := h.Presets.ListFraudPolicyPresets(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if presets == nil {
		presets = []FraudPolicyPresetDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, presets)
}
