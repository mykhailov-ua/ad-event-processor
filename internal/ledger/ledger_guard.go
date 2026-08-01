package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

const ledgerMarginWindow = time.Hour

func (w *Worker) runLedgerMarginCycle(ctx context.Context) error {
	if w == nil || w.pool == nil {
		return nil
	}
	start := time.Now()
	policies, err := w.fetchActivePolicies(ctx)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if err := w.evaluateLedgerMargin(ctx, policy); err != nil {
			slog.Error("ledger margin evaluation failed", "campaign_id", policy.CampaignID, "error", err)
		}
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		slog.Warn("ledger margin guard cycle slow", "duration_ms", elapsed.Milliseconds())
	}
	return nil
}

func (w *Worker) evaluateLedgerMargin(ctx context.Context, policy *Policy) error {
	if policy == nil {
		return nil
	}
	windowStart := time.Now().Add(-ledgerMarginWindow)
	sums, err := db.New(w.pool).SumCampaignMarginWindow(ctx, db.SumCampaignMarginWindowParams{
		CampaignID: pgtype.UUID{Bytes: policy.CampaignID, Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("sum margin window: %w", err)
	}
	if sums.AdvertiserSpendMicro <= 0 || sums.RtbCostMicro <= 0 {
		return nil
	}

	thresholdBps := CostOverRevenueThresholdBps(policy, w.cfg)
	limitMicro := CostOverRevenueLimitMicro(sums.AdvertiserSpendMicro, thresholdBps)
	if sums.RtbCostMicro <= limitMicro {
		return nil
	}

	var exists bool
	err = w.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM margin_guard_activity
			WHERE campaign_id = $1 AND action = 'pause' AND placement_id = ''
			  AND created_at > now() - INTERVAL '1 hour'
		)`, policy.CampaignID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	reason := fmt.Sprintf(
		"FORCE_PAUSE: rtb_cost %d exceeds revenue %d with threshold %d bps (limit %d)",
		sums.RtbCostMicro, sums.AdvertiserSpendMicro, thresholdBps, limitMicro,
	)
	type forcePauseMetrics struct {
		AdvertiserSpendMicro int64  `json:"advertiser_spend_micro"`
		RtbCostMicro         int64  `json:"rtb_cost_micro"`
		OperatorMarginMicro  int64  `json:"operator_margin_micro"`
		PublisherPayoutMicro int64  `json:"publisher_payout_micro"`
		ThresholdBps         int    `json:"threshold_bps"`
		WindowStart          string `json:"window_start"`
	}
	metricsJSON, err := json.Marshal(forcePauseMetrics{
		AdvertiserSpendMicro: sums.AdvertiserSpendMicro,
		RtbCostMicro:         sums.RtbCostMicro,
		OperatorMarginMicro:  sums.OperatorMarginMicro,
		PublisherPayoutMicro: sums.PublisherPayoutMicro,
		ThresholdBps:         thresholdBps,
		WindowStart:          windowStart.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal margin metrics: %w", err)
	}
	_, err = w.pool.Exec(ctx, `
		INSERT INTO margin_guard_activity (policy_id, campaign_id, placement_id, action, reason, metrics)
		VALUES ($1, $2, '', 'pause', $3, $4)`,
		policy.ID, policy.CampaignID, reason, metricsJSON)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]string{
		"campaign_id": policy.CampaignID.String(),
		"reason":      reason,
	})
	if err != nil {
		return fmt.Errorf("marshal pause payload: %w", err)
	}
	_, err = w.pool.Exec(ctx, `INSERT INTO outbox_events (event_type, payload) VALUES ($1, $2)`, "PAUSE_CAMPAIGN", payload)
	if err != nil {
		return err
	}

	slog.Info("margin guard ledger pause enqueued",
		"campaign_id", policy.CampaignID,
		"rtb_cost_micro", sums.RtbCostMicro,
		"advertiser_spend_micro", sums.AdvertiserSpendMicro,
		"threshold_bps", thresholdBps,
	)
	return nil
}
