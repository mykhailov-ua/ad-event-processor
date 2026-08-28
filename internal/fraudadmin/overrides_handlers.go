package fraudadmin

import (
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *HTTPHandlers) registerFraudOverrideRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, permAny func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Overrides == nil {
		return
	}
	mux.HandleFunc("POST /api/v1/fraud/overrides", limit(permAny([]string{"audit:write", "campaigns:write", "shards:write"}, h.postFraudOverride)))
}

func (h *HTTPHandlers) postFraudOverride(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveCustomerID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[FraudOverrideRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.IPHash != nil {
		normalized := strings.TrimSpace(strings.ToLower(*req.IPHash))
		req.IPHash = &normalized
		if normalized != "" && !ValidMLIPHashHex(normalized) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
			return
		}
	}
	if req.CampaignID != nil {
		raw := strings.TrimSpace(*req.CampaignID)
		if raw != "" {
			campID, err := uuid.Parse(raw)
			if err != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
				return
			}
			if h.AuthorizeCampaignAccess != nil {
				if err := h.AuthorizeCampaignAccess(r, campID); err != nil {
					h.writeServiceError(w, err)
					return
				}
			}
			req.CampaignID = &raw
		}
	}
	if err := h.Overrides.ApplyFraudScoringOverrideForCustomer(r.Context(), customerID, req); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
