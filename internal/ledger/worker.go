package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/notify"
	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignEntitlementRegistry interface {
	GetCampaign(id uuid.UUID) (*domain.Campaign, bool)
	GetEntitlements(customerID uuid.UUID) (licensing.Entitlements, bool)
}

type Worker struct {
	pool     *pgxpool.Pool
	ch       *database.CHQuery
	cfg      *config.Config
	registry CampaignEntitlementRegistry
	notifier notify.NotifierAPI
}

func NewWorker(pool *pgxpool.Pool, ch *database.CHQuery, cfg *config.Config, registry CampaignEntitlementRegistry, notifier notify.NotifierAPI) *Worker {
	return &Worker{
		pool:     pool,
		ch:       ch,
		cfg:      cfg,
		registry: registry,
		notifier: notifier,
	}
}

type PausePlacementPayload struct {
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id"`
}

const (
	marginGuardPlacementStatsQuery = `
SELECT
    campaign_id,
    placement_id,
    sum(spend_micro) AS spend,
    sum(revenue_micro) AS revenue,
    sum(click_count) AS clicks,
    sum(conversion_count) AS conversions
FROM placement_stats_hourly
WHERE campaign_id IN (?)
  AND hour >= subtractHours(now(), 24)
GROUP BY campaign_id, placement_id`

	marginGuardActivityInsertSQL = `
INSERT INTO margin_guard_activity (policy_id, campaign_id, placement_id, action, reason, metrics)
VALUES ($1, $2, $3, $4, $5, $6)`

	marginGuardOutboxInsertSQL = `
INSERT INTO outbox_events (event_type, payload) VALUES ($1, $2)`
)

func (w *Worker) Start(ctx context.Context, interval time.Duration) {
	slog.Info("margin guard worker starting", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunCycle(ctx); err != nil {
				slog.Error("margin guard cycle failed", "error", err)
			}
		}
	}
}

func (w *Worker) RunCycle(ctx context.Context) error {
	if err := w.runLedgerMarginCycle(ctx); err != nil {
		return err
	}
	if w.ch == nil {
		return nil
	}

	var lag int64
	chCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := w.ch.QueryRow(chCtx, "SELECT dateDiff('second', max(hour), now()) FROM placement_stats_hourly").Scan(&lag)
	if err != nil {
		_ = w.ch.QueryRow(chCtx, "SELECT dateDiff('second', max(snapshot_hour), now()) FROM cost_snapshots").Scan(&lag)
	}

	if lag > 300 {
		slog.Warn("margin guard skipping cycle due to ch lag", "lag_seconds", lag)
		return nil
	}

	policies, err := w.fetchActivePolicies(ctx)
	if err != nil {
		return fmt.Errorf("fetch policies: %w", err)
	}

	return w.evaluatePoliciesBatch(ctx, policies)
}

func (w *Worker) fetchActivePolicies(ctx context.Context) ([]*Policy, error) {
	rows, err := w.pool.Query(ctx, "SELECT id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active FROM margin_guard_policies WHERE is_active = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*Policy
	for rows.Next() {
		p := &Policy{}
		if err := rows.Scan(&p.ID, &p.CampaignID, &p.Name, &p.MinClicks, &p.RoiFloorPct, &p.ZeroConvStreak, &p.CostOverRevenueThresholdBps, &p.IsActive); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (w *Worker) policyEntitled(policy *Policy) bool {
	if w.registry == nil {
		return true
	}
	camp, ok := w.registry.GetCampaign(policy.CampaignID)
	if !ok {
		return true
	}
	ent, ok := w.registry.GetEntitlements(camp.CustomerID)
	if !ok {
		return true
	}
	return ent.Features.MarginGuard
}

func (w *Worker) evaluatePoliciesBatch(ctx context.Context, policies []*Policy) error {
	eligible := make([]*Policy, 0, len(policies))
	campaignIDs := make([]uuid.UUID, 0, len(policies))
	seenCampaign := make(map[uuid.UUID]struct{}, len(policies))
	for _, policy := range policies {
		if !w.policyEntitled(policy) {
			slog.Warn("skipping policy evaluation: customer not entitled to margin guard",
				"policy_id", policy.ID, "campaign_id", policy.CampaignID)
			continue
		}
		eligible = append(eligible, policy)
		if _, ok := seenCampaign[policy.CampaignID]; !ok {
			seenCampaign[policy.CampaignID] = struct{}{}
			campaignIDs = append(campaignIDs, policy.CampaignID)
		}
	}
	if len(eligible) == 0 {
		return nil
	}

	statsByCampaign, err := w.queryPlacementStatsBatch(ctx, campaignIDs)
	if err != nil {
		return err
	}

	decisions := make([]*Decision, 0)
	for _, policy := range eligible {
		for _, stats := range statsByCampaign[policy.CampaignID] {
			if decision, trigger := Evaluate(policy, &stats); trigger {
				decisions = append(decisions, decision)
			}
		}
	}
	return w.applyDecisionsBatch(ctx, decisions)
}

func (w *Worker) queryPlacementStatsBatch(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID][]PlacementStats, error) {
	out := make(map[uuid.UUID][]PlacementStats)
	if len(campaignIDs) == 0 {
		return out, nil
	}
	chCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := w.ch.Query(chCtx, marginGuardPlacementStatsQuery, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stats PlacementStats
		if err := rows.Scan(&stats.CampaignID, &stats.PlacementID, &stats.SpendMicro, &stats.RevenueMicro, &stats.Clicks, &stats.Conversions); err != nil {
			return nil, err
		}
		out[stats.CampaignID] = append(out[stats.CampaignID], stats)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type marginGuardPauseKey struct {
	campaignID uuid.UUID
	placement  string
}

func (w *Worker) applyDecisionsBatch(ctx context.Context, decisions []*Decision) error {
	if len(decisions) == 0 {
		return nil
	}

	pauseDecisions := make([]*Decision, 0, len(decisions))
	campaignIDSet := make(map[uuid.UUID]struct{})
	for _, d := range decisions {
		if d.Action != ActionPause {
			continue
		}
		pauseDecisions = append(pauseDecisions, d)
		campaignIDSet[d.CampaignID] = struct{}{}
	}
	if len(pauseDecisions) == 0 {
		return nil
	}

	campaignIDs := make([]uuid.UUID, 0, len(campaignIDSet))
	for id := range campaignIDSet {
		campaignIDs = append(campaignIDs, id)
	}

	existing := make(map[marginGuardPauseKey]struct{})
	rows, err := w.pool.Query(ctx, `
SELECT campaign_id, placement_id FROM margin_guard_activity
WHERE action = 'pause' AND created_at > now() - interval '1 day'
  AND campaign_id = ANY($1)`, campaignIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var campaignID uuid.UUID
		var placementID string
		if err := rows.Scan(&campaignID, &placementID); err != nil {
			return err
		}
		existing[marginGuardPauseKey{campaignID: campaignID, placement: placementID}] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	notify := make([]*Decision, 0, len(pauseDecisions))
	for _, d := range pauseDecisions {
		key := marginGuardPauseKey{campaignID: d.CampaignID, placement: d.PlacementID}
		if _, ok := existing[key]; ok {
			continue
		}
		metricsJSON, _ := json.Marshal(d.Metrics)
		batch.Queue(marginGuardActivityInsertSQL, d.PolicyID, d.CampaignID, d.PlacementID, d.Action, d.Reason, metricsJSON)
		payload, _ := json.Marshal(PausePlacementPayload{
			CampaignID:  d.CampaignID.String(),
			PlacementID: d.PlacementID,
		})
		batch.Queue(marginGuardOutboxInsertSQL, "PAUSE_PLACEMENT", payload)
		notify = append(notify, d)
	}
	if batch.Len() == 0 {
		return nil
	}

	br := w.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("margin guard batch item %d: %w", i, err)
		}
	}

	for _, d := range notify {
		w.notifyMarginGuardPause(ctx, d)
		slog.Info("margin guard applied decision",
			"campaign_id", d.CampaignID,
			"placement_id", d.PlacementID,
			"action", d.Action,
			"reason", d.Reason,
		)
	}
	return nil
}

func (w *Worker) notifyMarginGuardPause(ctx context.Context, d *Decision) {
	if w.notifier == nil {
		return
	}
	title := fmt.Sprintf("Margin Guard: Placement Paused (%s)", d.PlacementID)
	body := fmt.Sprintf("Campaign: %s\nPlacement: %s\nReason: %s\nROI: %.2f%%\nSpend: %s USD\nRevenue: %s USD\nClicks: %d\nConversions: %d",
		d.CampaignID, d.PlacementID, d.Reason, d.Metrics.RoiPct,
		money.FormatDecimal(d.Metrics.SpendMicro),
		money.FormatDecimal(d.Metrics.RevenueMicro),
		d.Metrics.Clicks, d.Metrics.Conversions)
	if _, err := w.notifier.SendNotification(ctx, "TELEGRAM", "admin", title, body); err != nil {
		slog.Error("failed to send margin guard notification", "error", err)
	}
}
