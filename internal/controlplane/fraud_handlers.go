package controlplane

import (
	"context"
	"net/http"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type CampaignFraudConfigDTO struct {
	CampaignID            string `json:"campaign_id"`
	FraudThresholdPass    uint8  `json:"fraud_threshold_pass"`
	FraudThresholdSuspect uint8  `json:"fraud_threshold_suspect"`
	FraudThresholdIVT     uint8  `json:"fraud_threshold_ivt"`
	FraudThresholdBlock   uint8  `json:"fraud_threshold_block"`
	GhostIVTEnabled       bool   `json:"ghost_ivt_enabled"`
	BehaviorFlags         uint32 `json:"behavior_flags"`
}

type PatchCampaignFraudRequest struct {
	Preset                *string `json:"preset,omitempty"`
	FraudThresholdPass    *uint8  `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect *uint8  `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT     *uint8  `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock   *uint8  `json:"fraud_threshold_block,omitempty"`
	GhostIVTEnabled       *bool   `json:"ghost_ivt_enabled,omitempty"`
	BehaviorFlags         *uint32 `json:"behavior_flags,omitempty"`
}

type CampaignFraudService interface {
	GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error)
	UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error)
	PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error)
}

func (campaigns *CampaignsHTTPHandlers) registerCampaignFraudRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if campaigns == nil || campaigns.CampaignFraud == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/fraud", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, campaigns.getCampaignFraud)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/fraud", limit(perm([]string{"campaigns:write"}, campaigns.patchCampaignFraud)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/fraud/preview", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, campaigns.postCampaignFraudPreview)))
}

func (campaigns *CampaignsHTTPHandlers) getCampaignFraud(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	cfg, err := campaigns.CampaignFraud.GetCampaignFraudConfig(r.Context(), campaignID)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, cfg)
}

func (campaigns *CampaignsHTTPHandlers) patchCampaignFraud(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[PatchCampaignFraudRequest](body)
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
	updated, err := campaigns.CampaignFraud.UpdateCampaignFraudConfig(r.Context(), campaignID, req)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (campaigns *CampaignsHTTPHandlers) postCampaignFraudPreview(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := campaigns.parseCampaignID(w, r)
	if !ok {
		return
	}
	if campaigns.AllowFraudPreview != nil && !campaigns.AllowFraudPreview(campaignID.String()) {
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
	out, err := campaigns.CampaignFraud.PreviewCampaignFraudImpact(r.Context(), campaignID, req)
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}
