package fraudadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/fraud"

	"github.com/google/uuid"
)

const (
	fraudPreviewClickHouseQueryTimeout = 10 * time.Second
	fraudPreviewLookbackDays           = 7
	fraudPreviewSampleLimit            = 10000
)

const fraudPreviewDisclaimer = "Estimate based on last 7d shadow scores; proxy-label tiers only (no policy replay)."

func resolveProposedFraudThresholds(ctx context.Context, host CampaignConfigHost, current campaignFraudThresholds, req campaign.PreviewCampaignFraudRequest) (campaignFraudThresholds, error) {
	pass := current.pass
	suspect := current.suspect
	ivt := current.ivt
	block := current.block

	if req.Preset != nil {
		presetPass, presetSuspect, presetIVT, presetBlock, err := host.ConfigResolvePresetThresholds(ctx, *req.Preset)
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

func countFraudPreviewTiers(scores []float64, thresholds campaignFraudThresholds) (campaign.FraudPreviewTierCountsDTO, int64) {
	var counts campaign.FraudPreviewTierCountsDTO
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

func queryCampaignShadowScores(ctx context.Context, host CampaignConfigHost, campaignID uuid.UUID) ([]float64, error) {
	ch := host.ConfigClickHouse()
	if ch == nil {
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

	rows, err := ch.Query(
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

func PreviewCampaignFraudImpact(ctx context.Context, host CampaignConfigHost, campaignID uuid.UUID, req campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error) {
	if campaignID == uuid.Nil {
		return campaign.CampaignFraudPreviewDTO{}, ValidationError("campaign_id is required")
	}

	current, err := getCampaignFraudThresholds(ctx, host, campaignID)
	if err != nil {
		return campaign.CampaignFraudPreviewDTO{}, err
	}
	proposed, err := resolveProposedFraudThresholds(ctx, host, current, req)
	if err != nil {
		return campaign.CampaignFraudPreviewDTO{}, err
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, fraudPreviewClickHouseQueryTimeout)
	defer cancel()

	scores, err := queryCampaignShadowScores(clickhouseCtx, host, campaignID)
	if err != nil {
		return campaign.CampaignFraudPreviewDTO{}, err
	}

	byTier, affected := countFraudPreviewTiers(scores, proposed)
	return campaign.CampaignFraudPreviewDTO{
		CampaignID:    campaignID.String(),
		AffectedIPs7d: affected,
		SampleSize:    int64(len(scores)),
		ByTier:        byTier,
		Disclaimer:    fraudPreviewDisclaimer,
	}, nil
}

func getCampaignFraudThresholds(ctx context.Context, host CampaignConfigHost, campaignID uuid.UUID) (campaignFraudThresholds, error) {
	cfg, err := GetCampaignFraudConfig(ctx, host, campaignID)
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
