package opsadmin

import (
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) RegisterFraudPresetOpsRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.FraudPresets == nil {
		return
	}
	mux.HandleFunc("PATCH /api/v1/ops/fraud/presets/{name}", limit(perm("shards:write", h.patchFraudPolicyPreset)))
}

func (h *HTTPHandlers) patchFraudPolicyPreset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(strings.ToLower(r.PathValue("name")))
	if name == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "preset name is required")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[PatchFraudPolicyPresetRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Pass == nil && req.Suspect == nil && req.IVT == nil && req.Block == nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "at least one threshold field is required")
		return
	}
	out, err := h.FraudPresets.UpdateFraudPolicyPreset(r.Context(), name, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
