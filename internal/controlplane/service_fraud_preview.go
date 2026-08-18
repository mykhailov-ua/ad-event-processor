package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/fraud"

	"github.com/google/uuid"
)

const (
	fraudPreviewCHQueryTimeout = 10 * time.Second
	fraudPreviewLookbackDays   = 7
	fraudPreviewSampleLimit    = 10000
)

const fraudPreviewDisclaimer = "Estimate based on last 7d shadow scores; proxy-label tiers only (no policy replay)."

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

func (s *Service) resolveProposedFraudThresholds(ctx context.Context, current campaignFraudThresholds, req PreviewCampaignFraudRequest) (campaignFraudThresholds, error) {
	pass := current.pass
	suspect := current.suspect
	ivt := current.ivt
	block := current.block

	if req.Preset != nil {
		presetPass, presetSuspect, presetIVT, presetBlock, err := s.resolveFraudPresetThresholds(ctx, *req.Preset)
		if err != nil {
			return campaignFraudThresholds{}, err
		}
		pass = presetPass
		suspect = presetSuspect
		ivt = presetIVT
		block = presetBlock
	}
	if req.FraudThresholdPass != nil {
		pass = *req.FraudThresholdPass
	}
	if req.FraudThresholdSuspect != nil {
		suspect = *req.FraudThresholdSuspect
	}
	if req.FraudThresholdIVT != nil {
		ivt = *req.FraudThresholdIVT
	}
	if req.FraudThresholdBlock != nil {
		block = *req.FraudThresholdBlock
	}
	if err := validateFraudThresholds(pass, suspect, ivt, block); err != nil {
		return campaignFraudThresholds{}, err
	}
	return campaignFraudThresholds{pass: pass, suspect: suspect, ivt: ivt, block: block}, nil
}

func countFraudPreviewTiers(scores []float64, thresholds campaignFraudThresholds) (FraudPreviewTierCountsDTO, int64) {
	var counts FraudPreviewTierCountsDTO
	var affected int64
	for _, mlScore := range scores {
		tier, _ := fraud.MapProbabilityTierWithThresholds(mlScore, thresholds.pass, thresholds.suspect, thresholds.ivt, thresholds.block)
		switch tier {
		case fraud.FraudTierSuspect:
			counts.Suspect++
			affected++
		case fraud.FraudTierIVT:
			counts.IVT++
			affected++
		case fraud.FraudTierBlock:
			counts.Block++
			affected++
		}
	}
	return counts, affected
}

func (s *Service) queryCampaignShadowScores(ctx context.Context, campaignID uuid.UUID) ([]float64, error) {
	if s == nil || s.chQuery == nil {
		return nil, fmt.Errorf("clickhouse not configured")
	}

	query := `
SELECT
    argMax(s.score, s.created_at) AS ml_score
FROM ml_shadow_scores AS s
INNER JOIN (
    SELECT ip_hash
    FROM ml_features_1m
    WHERE campaign_id = ?
      AND window_start >= subtractDays(now(), ?)
    GROUP BY ip_hash
    LIMIT ?
) AS f ON f.ip_hash = s.ip_hash
WHERE s.created_at >= subtractDays(now(), ?)
GROUP BY s.ip_hash
LIMIT ?`

	rows, err := s.chQuery.Query(
		ctx,
		query,
		campaignID,
		fraudPreviewLookbackDays,
		fraudPreviewSampleLimit,
		fraudPreviewLookbackDays,
		fraudPreviewSampleLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("fraud preview clickhouse query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]float64, 0, 256)
	for rows.Next() {
		var mlScore float64
		if err := rows.Scan(&mlScore); err != nil {
			return nil, fmt.Errorf("fraud preview clickhouse scan: %w", err)
		}
		out = append(out, mlScore)
	}
	return out, rows.Err()
}

func (s *Service) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error) {
	if campaignID == uuid.Nil {
		return CampaignFraudPreviewDTO{}, errValidation("campaign_id is required")
	}

	current, err := s.getCampaignFraudThresholds(ctx, campaignID)
	if err != nil {
		return CampaignFraudPreviewDTO{}, err
	}
	proposed, err := s.resolveProposedFraudThresholds(ctx, current, req)
	if err != nil {
		return CampaignFraudPreviewDTO{}, err
	}

	chCtx, cancel := context.WithTimeout(ctx, fraudPreviewCHQueryTimeout)
	defer cancel()

	scores, err := s.queryCampaignShadowScores(chCtx, campaignID)
	if err != nil {
		return CampaignFraudPreviewDTO{}, err
	}

	byTier, affected := countFraudPreviewTiers(scores, proposed)
	return CampaignFraudPreviewDTO{
		CampaignID:    campaignID.String(),
		AffectedIPs7d: affected,
		SampleSize:    int64(len(scores)),
		ByTier:        byTier,
		Disclaimer:    fraudPreviewDisclaimer,
	}, nil
}

func (s *Service) getCampaignFraudThresholds(ctx context.Context, campaignID uuid.UUID) (campaignFraudThresholds, error) {
	cfg, err := s.GetCampaignFraudConfig(ctx, campaignID)
	if err != nil {
		return campaignFraudThresholds{}, err
	}
	return campaignFraudThresholds{
		pass:    cfg.FraudThresholdPass,
		suspect: cfg.FraudThresholdSuspect,
		ivt:     cfg.FraudThresholdIVT,
		block:   cfg.FraudThresholdBlock,
	}, nil
}
