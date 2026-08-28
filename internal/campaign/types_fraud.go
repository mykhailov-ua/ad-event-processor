package campaign

import (
	"context"
	"encoding/json"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type CampaignFraudConfigDTO struct {
	CampaignID               string                       `json:"campaign_id"`
	FraudThresholdPass       uint8                        `json:"fraud_threshold_pass"`
	FraudThresholdSuspect    uint8                        `json:"fraud_threshold_suspect"`
	FraudThresholdIVT        uint8                        `json:"fraud_threshold_ivt"`
	FraudThresholdBlock      uint8                        `json:"fraud_threshold_block"`
	SilentRejectEnabled      bool                         `json:"silent_reject_enabled"`
	BehaviorFlags            uint32                       `json:"behavior_flags"`
	CanvasRetestEnabled      bool                         `json:"canvas_retest_enabled"`
	CgnatIPPolicyEnabled     bool                         `json:"cgnat_ip_policy_enabled"`
	AcceptLangGeoEnabled     bool                         `json:"accept_lang_geo_enabled"`
	JSONSerializationEnabled bool                         `json:"json_serialization_enabled"`
	ConversionRejectRules    domain.ConversionRejectRules `json:"conversion_reject_rules"`
}

type PatchCampaignFraudRequest struct {
	Preset                   *string                       `json:"preset,omitempty"`
	FraudThresholdPass       *uint8                        `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect    *uint8                        `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT        *uint8                        `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock      *uint8                        `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled      *bool                         `json:"silent_reject_enabled,omitempty"`
	BehaviorFlags            *uint32                       `json:"behavior_flags,omitempty"`
	CanvasRetestEnabled      *bool                         `json:"canvas_retest_enabled,omitempty"`
	CgnatIPPolicyEnabled     *bool                         `json:"cgnat_ip_policy_enabled,omitempty"`
	AcceptLangGeoEnabled     *bool                         `json:"accept_lang_geo_enabled,omitempty"`
	JSONSerializationEnabled *bool                         `json:"json_serialization_enabled,omitempty"`
	ConversionRejectRules    *domain.ConversionRejectRules `json:"conversion_reject_rules,omitempty"`
}

type patchCampaignFraudRequestRaw struct {
	Preset                   *string                       `json:"preset,omitempty"`
	FraudThresholdPass       *uint8                        `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect    *uint8                        `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT        *uint8                        `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock      *uint8                        `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled      *bool                         `json:"silent_reject_enabled,omitempty"`
	SilentRejectPatchLegacy  *bool                         `json:"ghost_ivt_enabled,omitempty"`
	BehaviorFlags            *uint32                       `json:"behavior_flags,omitempty"`
	CanvasRetestEnabled      *bool                         `json:"canvas_retest_enabled,omitempty"`
	CgnatIPPolicyEnabled     *bool                         `json:"cgnat_ip_policy_enabled,omitempty"`
	AcceptLangGeoEnabled     *bool                         `json:"accept_lang_geo_enabled,omitempty"`
	JSONSerializationEnabled *bool                         `json:"json_serialization_enabled,omitempty"`
	ConversionRejectRules    *domain.ConversionRejectRules `json:"conversion_reject_rules,omitempty"`
}

func decodePatchCampaignFraudRequest(body []byte) (PatchCampaignFraudRequest, error) {
	var raw patchCampaignFraudRequestRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return PatchCampaignFraudRequest{}, err
	}
	req := PatchCampaignFraudRequest{
		Preset:                raw.Preset,
		FraudThresholdPass:    raw.FraudThresholdPass,
		FraudThresholdSuspect: raw.FraudThresholdSuspect,
		FraudThresholdIVT:     raw.FraudThresholdIVT,
		FraudThresholdBlock:   raw.FraudThresholdBlock,
		BehaviorFlags:         raw.BehaviorFlags,
	}
	if raw.CanvasRetestEnabled != nil {
		req.CanvasRetestEnabled = raw.CanvasRetestEnabled
	}
	if raw.CgnatIPPolicyEnabled != nil {
		req.CgnatIPPolicyEnabled = raw.CgnatIPPolicyEnabled
	}
	if raw.AcceptLangGeoEnabled != nil {
		req.AcceptLangGeoEnabled = raw.AcceptLangGeoEnabled
	}
	if raw.JSONSerializationEnabled != nil {
		req.JSONSerializationEnabled = raw.JSONSerializationEnabled
	}
	if raw.ConversionRejectRules != nil {
		req.ConversionRejectRules = raw.ConversionRejectRules
	}
	if raw.SilentRejectEnabled != nil {
		req.SilentRejectEnabled = raw.SilentRejectEnabled
	} else if raw.SilentRejectPatchLegacy != nil {
		req.SilentRejectEnabled = raw.SilentRejectPatchLegacy
	}
	return req, nil
}

type CampaignFraudPreviewDTO struct {
	CampaignID    string                    `json:"campaign_id"`
	AffectedIPs7d int64                     `json:"affected_ips_7d"`
	SampleSize    int64                     `json:"sample_size"`
	ByTier        FraudPreviewTierCountsDTO `json:"by_tier"`
	Disclaimer    string                    `json:"disclaimer"`
}

type FraudPreviewTierCountsDTO struct {
	Suspect int64 `json:"suspect"`
	IVT     int64 `json:"ivt"`
	Block   int64 `json:"block"`
}

type PreviewCampaignFraudRequest struct {
	Preset                *string `json:"preset,omitempty"`
	FraudThresholdPass    *uint8  `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect *uint8  `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT     *uint8  `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock   *uint8  `json:"fraud_threshold_block,omitempty"`
}

type CampaignFraudService interface {
	GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error)
	UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error)
	PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error)
}
