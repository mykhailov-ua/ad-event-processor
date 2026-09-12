package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type campaignMarginWindowSums struct {
	advertiserSpendMicro int64
	rtbCostMicro         int64
	operatorMarginMicro  int64
	publisherPayoutMicro int64
}

func (w *Worker) evaluateLedgerMarginBatch(ctx context.Context, policies []*Policy) error {
	if len(policies) == 0 {
		return nil
	}
	if w.enforcement == nil {
		return fmt.Errorf("margin guard enforcement host not configured")
	}

	campaignIDs := make([]uuid.UUID, 0, len(policies))
	seen := make(map[uuid.UUID]struct{}, len(policies))
	for _, policy := range policies {
		if _, ok := seen[policy.CampaignID]; ok {
			continue
		}
		seen[policy.CampaignID] = struct{}{}
		campaignIDs = append(campaignIDs, policy.CampaignID)
	}

	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	windowStart := time.Now().Add(-ledgerMarginWindow).UTC()
	q := db.New(w.pool)
	sumRows, err := q.SumCampaignMarginWindowByCampaignIDs(ctx, db.SumCampaignMarginWindowByCampaignIDsParams{
		CampaignIds: pgIDs,
		WindowStart: pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("sum margin window batch: %w", err)
	}

	sumsByCampaign := make(map[uuid.UUID]campaignMarginWindowSums, len(sumRows))
	for _, row := range sumRows {
		id, err := uuid.FromBytes(row.CampaignID.Bytes[:])
		if err != nil {
			continue
		}
		sumsByCampaign[id] = campaignMarginWindowSums{
			advertiserSpendMicro: row.AdvertiserSpendMicro,
			rtbCostMicro:         row.RtbCostMicro,
			operatorMarginMicro:  row.OperatorMarginMicro,
			publisherPayoutMicro: row.PublisherPayoutMicro,
		}
	}

	type breachCandidate struct {
		policy *Policy
		sums   campaignMarginWindowSums
		limit  int64
		bps    int
	}
	candidates := make([]breachCandidate, 0)
	breachCampaignIDs := make([]pgtype.UUID, 0)
	for _, policy := range policies {
		sums := sumsByCampaign[policy.CampaignID]
		if sums.advertiserSpendMicro <= 0 || sums.rtbCostMicro <= 0 {
			continue
		}
		thresholdBps := CostOverRevenueThresholdBps(policy, w.cfg)
		limitMicro := CostOverRevenueLimitMicro(sums.advertiserSpendMicro, thresholdBps)
		if sums.rtbCostMicro <= limitMicro {
			continue
		}
		candidates = append(candidates, breachCandidate{
			policy: policy,
			sums:   sums,
			limit:  limitMicro,
			bps:    thresholdBps,
		})
		breachCampaignIDs = append(breachCampaignIDs, pgtype.UUID{Bytes: policy.CampaignID, Valid: true})
	}
	if len(candidates) == 0 {
		return nil
	}

	lastPauseByCampaign := make(map[uuid.UUID]time.Time)
	pauseRows, err := q.ListRecentMarginGuardPausesByCampaigns(ctx, breachCampaignIDs)
	if err != nil {
		return fmt.Errorf("list recent margin pauses: %w", err)
	}
	for _, row := range pauseRows {
		id, err := uuid.FromBytes(row.CampaignID.Bytes[:])
		if err != nil {
			continue
		}
		if row.LastPauseAt.Valid {
			lastPauseByCampaign[id] = row.LastPauseAt.Time
		}
	}

	now := time.Now()
	for _, item := range candidates {
		if lastPause, ok := lastPauseByCampaign[item.policy.CampaignID]; ok && WithinCooldown(lastPause, PolicyCooldownSec(item.policy), now) {
			continue
		}
		reason := fmt.Sprintf(
			"FORCE_PAUSE: rtb_cost %d exceeds revenue %d with threshold %d bps (limit %d)",
			item.sums.rtbCostMicro, item.sums.advertiserSpendMicro, item.bps, item.limit,
		)
		metricsJSON, err := json.Marshal(forcePauseMetrics{
			AdvertiserSpendMicro: item.sums.advertiserSpendMicro,
			RtbCostMicro:         item.sums.rtbCostMicro,
			OperatorMarginMicro:  item.sums.operatorMarginMicro,
			PublisherPayoutMicro: item.sums.publisherPayoutMicro,
			ThresholdBps:         item.bps,
			WindowStart:          windowStart.UTC().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("marshal margin metrics: %w", err)
		}
		_, err = w.pool.Exec(ctx, `
			INSERT INTO margin_guard_activity (policy_id, campaign_id, placement_id, action, reason, metrics)
			VALUES ($1, $2, '', 'pause', $3, $4)`,
			item.policy.ID, item.policy.CampaignID, reason, metricsJSON)
		if err != nil {
			return err
		}
		if err := w.applyCampaignBreach(ctx, item.policy, item.policy.CampaignID, reason); err != nil {
			return fmt.Errorf("apply campaign breach %s: %w", item.policy.CampaignID, err)
		}

		slog.Info("margin guard ledger pause applied",
			"campaign_id", item.policy.CampaignID,
			"rtb_cost_micro", item.sums.rtbCostMicro,
			"advertiser_spend_micro", item.sums.advertiserSpendMicro,
			"threshold_bps", item.bps,
		)
	}
	return nil
}

type forcePauseMetrics struct {
	AdvertiserSpendMicro int64  `json:"advertiser_spend_micro"`
	RtbCostMicro         int64  `json:"rtb_cost_micro"`
	OperatorMarginMicro  int64  `json:"operator_margin_micro"`
	PublisherPayoutMicro int64  `json:"publisher_payout_micro"`
	ThresholdBps         int    `json:"threshold_bps"`
	WindowStart          string `json:"window_start"`
}
