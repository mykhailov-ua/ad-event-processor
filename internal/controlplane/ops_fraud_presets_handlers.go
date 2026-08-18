package controlplane

import (
	"net/http"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
)

func (ops *OpsHTTPHandlers) registerFraudPresetOpsRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if ops == nil || ops.FraudPresets == nil {
		return
	}
	mux.HandleFunc("PATCH /api/v1/ops/fraud/presets/{name}", limit(perm("shards:write", ops.patchFraudPolicyPreset)))
}

func (ops *OpsHTTPHandlers) patchFraudPolicyPreset(w http.ResponseWriter, r *http.Request) {
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
	out, err := ops.FraudPresets.UpdateFraudPolicyPreset(r.Context(), name, req)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
