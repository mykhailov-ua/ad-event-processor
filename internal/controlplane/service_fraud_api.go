package controlplane

import (
	"context"

	"github.com/google/uuid"
)

func toAdminCampaignFraudConfig(cfg CampaignFraudConfigDTO) CampaignFraudConfigDTO {
	return CampaignFraudConfigDTO{
		CampaignID:            cfg.CampaignID,
		FraudThresholdPass:    cfg.FraudThresholdPass,
		FraudThresholdSuspect: cfg.FraudThresholdSuspect,
		FraudThresholdIVT:     cfg.FraudThresholdIVT,
		FraudThresholdBlock:   cfg.FraudThresholdBlock,
		GhostIVTEnabled:       cfg.GhostIVTEnabled,
		BehaviorFlags:         cfg.BehaviorFlags,
	}
}

func (s *Service) GetCampaignFraudConfigAPI(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error) {
	cfg, err := s.GetCampaignFraudConfig(ctx, campaignID)
	if err != nil {
		return CampaignFraudConfigDTO{}, err
	}
	return toAdminCampaignFraudConfig(cfg), nil
}

func (s *Service) UpdateCampaignFraudConfigAPI(ctx context.Context, campaignID uuid.UUID, req PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error) {
	upd := CampaignFraudConfigUpdate{
		Preset:                req.Preset,
		FraudThresholdPass:    req.FraudThresholdPass,
		FraudThresholdSuspect: req.FraudThresholdSuspect,
		FraudThresholdIVT:     req.FraudThresholdIVT,
		FraudThresholdBlock:   req.FraudThresholdBlock,
		GhostIVTEnabled:       req.GhostIVTEnabled,
		BehaviorFlags:         req.BehaviorFlags,
	}
	cfg, err := s.UpdateCampaignFraudConfig(ctx, campaignID, upd)
	if err != nil {
		return CampaignFraudConfigDTO{}, err
	}
	return toAdminCampaignFraudConfig(cfg), nil
}

type campaignFraudAPIAdapter struct {
	svc *Service
}

func (a campaignFraudAPIAdapter) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error) {
	return a.svc.GetCampaignFraudConfigAPI(ctx, campaignID)
}

func (a campaignFraudAPIAdapter) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error) {
	return a.svc.UpdateCampaignFraudConfigAPI(ctx, campaignID, req)
}

func (a campaignFraudAPIAdapter) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error) {
	return a.svc.PreviewCampaignFraudImpact(ctx, campaignID, req)
}
