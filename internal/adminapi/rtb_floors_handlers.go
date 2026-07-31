package adminapi

import (
	"context"
	"net/http"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
)

type RtbFloorSuggestionDTO struct {
	PlacementID         string  `json:"placement_id"`
	DealID              string  `json:"deal_id"`
	CurrentFloorMicro   int64   `json:"current_floor_micro"`
	SuggestedFloorMicro int64   `json:"suggested_floor_micro"`
	WinRate             float64 `json:"win_rate"`
	SampleN             int64   `json:"sample_n"`
	FloorBucketMicro    int64   `json:"floor_bucket_micro"`
	ComputedAt          string  `json:"computed_at"`
}

type RtbFloorsApplyRequest struct {
	PlacementIDs []string `json:"placement_ids,omitempty"`
}

type RtbFloorsApplyResult struct {
	DryRun      bool                    `json:"dry_run"`
	Applied     int                     `json:"applied"`
	Suggestions []RtbFloorSuggestionDTO `json:"suggestions"`
	OutboxRows  int                     `json:"outbox_rows"`
}

type RtbFloorOptimizer interface {
	ApplyRtbFloorSuggestions(ctx context.Context, dryRun bool, placementIDs []string) (RtbFloorsApplyResult, error)
}

type RtbFloorsHTTPHandlers struct {
	Service           RtbFloorOptimizer
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
}

func (rtbFloors *RtbFloorsHTTPHandlers) Register(mux *http.ServeMux) {
	if rtbFloors == nil || rtbFloors.Service == nil {
		return
	}
	limit := rtbFloors.ApplyRateLimit
	perm := rtbFloors.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/rtb/floors/apply", limit(perm("settings:write", rtbFloors.applyFloors)))
}

func (rtbFloors *RtbFloorsHTTPHandlers) applyFloors(w http.ResponseWriter, r *http.Request) {
	dryRun := r.URL.Query().Get("dry_run") == "1" || r.URL.Query().Get("dry_run") == "true"
	var req RtbFloorsApplyRequest
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if len(body) > 0 {
		decoded, decodeErr := coldpath.DecodeBody[RtbFloorsApplyRequest](body)
		if decodeErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
			return
		}
		req = decoded
	}
	result, err := rtbFloors.Service.ApplyRtbFloorSuggestions(r.Context(), dryRun, req.PlacementIDs)
	if err != nil {
		if rtbFloors.WriteServiceError != nil {
			rtbFloors.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
