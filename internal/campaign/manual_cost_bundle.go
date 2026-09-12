package campaign

import (
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *CampaignsHTTPHandlers) registerManualCostRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || mux == nil {
		return
	}
	mux.HandleFunc("PUT /api/v1/campaigns/{id}/manual-cost", limit(perm([]string{"campaigns:write"}, h.putManualCampaignCost)))
}

func (h *CampaignsHTTPHandlers) putManualCampaignCost(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	if h.PostgresPool == nil {
		h.WriteServiceError(w, errServiceUnavailable())
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ManualCampaignCostRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	customerID, err := h.ResolveCustomerID(r, nil)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	resp, err := PutManualCampaignCost(r.Context(), h.PostgresPool, customerID, campaignID, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
