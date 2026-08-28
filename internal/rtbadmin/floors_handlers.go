package rtbadmin

import (
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type FloorsHTTPHandlers struct {
	Service           FloorOptimizer
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
}

func (h *FloorsHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/rtb/floors/apply", limit(perm("settings:write", h.applyFloors)))
}

func (h *FloorsHTTPHandlers) applyFloors(w http.ResponseWriter, r *http.Request) {
	dryRun := r.URL.Query().Get("dry_run") == "1" || r.URL.Query().Get("dry_run") == "true"
	var req FloorsApplyRequest
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if len(body) > 0 {
		decoded, decodeErr := coldpath.DecodeBody[FloorsApplyRequest](body)
		if decodeErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
			return
		}
		req = decoded
	}
	result, err := h.Service.ApplyRtbFloorSuggestions(r.Context(), dryRun, req.PlacementIDs)
	if err != nil {
		if h.WriteServiceError != nil {
			h.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
