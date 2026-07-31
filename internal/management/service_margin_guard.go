package management

import (
	"context"
	"fmt"
	"time"

	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/internal/marginguard"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) CreateMarginGuardPolicy(ctx context.Context, p *marginguard.Policy) error {
	thresholdBps := marginguard.CostOverRevenueThresholdBps(p, s.cfg)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO margin_guard_policies (campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, ingestion.ToUUID(p.CampaignID), p.Name, p.MinClicks, p.RoiFloorPct, p.ZeroConvStreak, thresholdBps, p.IsActive)
	return err
}

func (s *Service) ListMarginGuardPolicies(ctx context.Context, campaignID uuid.UUID) ([]*marginguard.Policy, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active
		FROM margin_guard_policies
		WHERE campaign_id = $1
	`, ingestion.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*marginguard.Policy
	for rows.Next() {
		p := &marginguard.Policy{}
		if err := rows.Scan(&p.ID, &p.CampaignID, &p.Name, &p.MinClicks, &p.RoiFloorPct, &p.ZeroConvStreak, &p.CostOverRevenueThresholdBps, &p.IsActive); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func (s *Service) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (map[string]any, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	windowStart := time.Now().Add(-1 * time.Hour)
	q := db.New(s.pool)
	sums, err := q.SumCampaignMarginWindow(ctx, db.SumCampaignMarginWindowParams{
		CampaignID: ingestion.ToUUID(campaignID),
		CreatedAt:  pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	thresholdBps := marginguard.CostOverRevenueThresholdBps(nil, s.cfg)
	policies, err := s.ListMarginGuardPolicies(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(policies) > 0 {
		thresholdBps = marginguard.CostOverRevenueThresholdBps(policies[0], s.cfg)
	}
	limitMicro := marginguard.CostOverRevenueLimitMicro(sums.AdvertiserSpendMicro, thresholdBps)
	return map[string]any{
		"campaign_id":             campaignID.String(),
		"window_start":            windowStart.UTC().Format(time.RFC3339),
		"window_hours":            1,
		"advertiser_spend_micro":  sums.AdvertiserSpendMicro,
		"rtb_cost_micro":          sums.RtbCostMicro,
		"operator_margin_micro":   sums.OperatorMarginMicro,
		"publisher_payout_micro":  sums.PublisherPayoutMicro,
		"cost_over_revenue_limit": limitMicro,
		"threshold_bps":           thresholdBps,
		"margin_breach":           sums.RtbCostMicro > limitMicro && sums.AdvertiserSpendMicro > 0,
	}, nil
}

func (s *Service) GetMarginGuardActivity(ctx context.Context, campaignID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, policy_id, campaign_id, placement_id, action, reason, metrics, created_at
		FROM margin_guard_activity
		WHERE campaign_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, ingestion.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []map[string]any
	for rows.Next() {
		var id, policyID, campID uuid.UUID
		var placementID, action, reason string
		var metrics map[string]any
		var createdAt interface{}
		if err := rows.Scan(&id, &policyID, &campID, &placementID, &action, &reason, &metrics, &createdAt); err != nil {
			return nil, err
		}
		activities = append(activities, map[string]any{
			"id":           id,
			"policy_id":    policyID,
			"campaign_id":  campID,
			"placement_id": placementID,
			"action":       action,
			"reason":       reason,
			"metrics":      metrics,
			"created_at":   createdAt,
		})
	}
	return activities, nil
}

func (s *Service) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	payload, err := coldpath.MarshalJSON(PausePlacementPayload{
		CampaignID:  campaignID.String(),
		PlacementID: placementID,
		Action:      "remove",
	})
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload)
		VALUES ($1, $2)`, "PAUSE_PLACEMENT", payload)
	return err
}
