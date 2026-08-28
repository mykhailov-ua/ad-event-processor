package campaign

import (
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *CampaignsHTTPHandlers) registerCampaignFraudRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.CampaignFraud == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/fraud", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaignFraud)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/fraud", limit(perm([]string{"campaigns:write"}, h.patchCampaignFraud)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/fraud/preview", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.postCampaignFraudPreview)))
}

func (h *CampaignsHTTPHandlers) getCampaignFraud(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	cfg, err := h.CampaignFraud.GetCampaignFraudConfig(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, cfg)
}

func (h *CampaignsHTTPHandlers) patchCampaignFraud(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := decodePatchCampaignFraudRequest(body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Preset != nil {
		preset := strings.TrimSpace(strings.ToLower(*req.Preset))
		if preset == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "preset must not be empty")
			return
		}
		req.Preset = &preset
	}
	updated, err := h.CampaignFraud.UpdateCampaignFraudConfig(r.Context(), campaignID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *CampaignsHTTPHandlers) postCampaignFraudPreview(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.parseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AllowFraudPreview != nil && !h.AllowFraudPreview(campaignID.String()) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "fraud preview rate limit exceeded")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[PreviewCampaignFraudRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Preset != nil {
		preset := strings.TrimSpace(strings.ToLower(*req.Preset))
		if preset == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "preset must not be empty")
			return
		}
		req.Preset = &preset
	}
	out, err := h.CampaignFraud.PreviewCampaignFraudImpact(r.Context(), campaignID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
