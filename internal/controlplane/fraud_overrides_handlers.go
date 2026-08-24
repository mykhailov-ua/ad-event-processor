package controlplane

import (
	"context"
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (fraud *FraudHTTPHandlers) registerFraudOverrideRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, permAny func([]string, http.HandlerFunc) http.HandlerFunc) {
	if fraud == nil || fraud.Overrides == nil {
		return
	}
	mux.HandleFunc("POST /api/v1/fraud/overrides", limit(permAny([]string{"audit:write", "campaigns:write", "shards:write"}, fraud.postFraudOverride)))
}

func (fraud *FraudHTTPHandlers) postFraudOverride(w http.ResponseWriter, r *http.Request) {
	customerID, ok := fraud.resolveCustomerID(w, r)
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
		if normalized != "" && !validMLIPHashHex(normalized) {
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
			if fraud.AuthorizeCampaignAccess != nil {
				if err := fraud.AuthorizeCampaignAccess(r, campID); err != nil {
					fraud.writeServiceError(w, err)
					return
				}
			}
			req.CampaignID = &raw
		}
	}
	if err := fraud.Overrides.ApplyFraudScoringOverrideForCustomer(r.Context(), customerID, req); err != nil {
		fraud.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type fraudOverridesAPIAdapter struct {
	svc *Service
}

func (a fraudOverridesAPIAdapter) ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req FraudOverrideRequest) error {
	return a.svc.ApplyFraudScoringOverrideForCustomer(ctx, customerID, req)
}
