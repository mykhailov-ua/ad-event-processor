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

	windowStart := time.Now().Add(-ledgerMarginWindow)
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

	paused := make(map[uuid.UUID]struct{})
	pauseRows, err := q.ListRecentMarginGuardPausesByCampaigns(ctx, breachCampaignIDs)
	if err != nil {
		return fmt.Errorf("list recent margin pauses: %w", err)
	}
	for _, row := range pauseRows {
		id, err := uuid.FromBytes(row.Bytes[:])
		if err != nil {
			continue
		}
		paused[id] = struct{}{}
	}

	eventTypes := make([]string, 0, len(candidates))
	payloads := make([][]byte, 0, len(candidates))
	for _, item := range candidates {
		if _, ok := paused[item.policy.CampaignID]; ok {
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

		payload, err := json.Marshal(map[string]string{
			"campaign_id": item.policy.CampaignID.String(),
			"reason":      reason,
		})
		if err != nil {
			return fmt.Errorf("marshal pause payload: %w", err)
		}
		eventTypes = append(eventTypes, "PAUSE_CAMPAIGN")
		payloads = append(payloads, payload)

		slog.Info("margin guard ledger pause enqueued",
			"campaign_id", item.policy.CampaignID,
			"rtb_cost_micro", item.sums.rtbCostMicro,
			"advertiser_spend_micro", item.sums.advertiserSpendMicro,
			"threshold_bps", item.bps,
		)
	}
	if len(eventTypes) == 0 {
		return nil
	}
	return q.CreateOutboxEventsBatch(ctx, db.CreateOutboxEventsBatchParams{
		EventTypes: eventTypes,
		Payloads:   payloads,
	})
}

type forcePauseMetrics struct {
	AdvertiserSpendMicro int64  `json:"advertiser_spend_micro"`
	RtbCostMicro         int64  `json:"rtb_cost_micro"`
	OperatorMarginMicro  int64  `json:"operator_margin_micro"`
	PublisherPayoutMicro int64  `json:"publisher_payout_micro"`
	ThresholdBps         int    `json:"threshold_bps"`
	WindowStart          string `json:"window_start"`
}
