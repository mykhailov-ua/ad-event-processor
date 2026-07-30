package management

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"espx/internal/database"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/internal/metrics"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

type OutboxWorker struct {
	svc *Service
}

func NewOutboxWorker(svc *Service) *OutboxWorker {
	return &OutboxWorker{svc: svc}
}

type CampaignPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}

type SettingsPayload struct {
	Settings map[string]string `json:"settings"`
}

type BlacklistPayload struct {
	Action string `json:"action"`
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

type FraudThreatPayload struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

func normalizeBlacklistReason(reason string) string {
	if reason == "" {
		return "manual"
	}
	return reason
}

func (w *OutboxWorker) Start(ctx context.Context, interval time.Duration) {
	if err := w.ProcessOutbox(ctx); err != nil {
		slog.Error("outbox startup cold sync failed", "error", err)
	}

	slog.Info("outbox worker starting polling loop", "interval", interval)

	pollBackoff := newOutboxPollBackoff()
	pollTimer := time.NewTimer(interval)
	defer pollTimer.Stop()

	recoveryTicker := time.NewTicker(interval * 5)
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-recoveryTicker.C:
			w.reclaimStaleProcessing(ctx)
			w.recordOutboxLagMetrics(ctx)
		case <-pollTimer.C:
			var processed int
			var err error
			if w.svc != nil {
				err = w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
					var innerErr error
					processed, innerErr = w.ProcessOutboxWithCount(runCtx, 1000)
					return innerErr
				})
			} else {
				processed, err = w.ProcessOutboxWithCount(ctx, 1000)
			}
			w.recordOutboxLagMetrics(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("outbox polling loop iteration failed, retrying in 2s", "error", err)
				pollTimer.Reset(2 * time.Second)
				continue
			}

			pollTimer.Reset(pollBackoff.next(processed))
		}
	}
}

func (w *OutboxWorker) reclaimStaleProcessing(ctx context.Context) {
	_, err := w.svc.GetPool().Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PENDING', processing_started_at = NULL
		WHERE status = 'PROCESSING'
		  AND processing_started_at IS NOT NULL
		  AND processing_started_at < NOW() - INTERVAL '1 minute'`)
	if err != nil && ctx.Err() == nil && !database.IsShutdownError(err) {
		slog.Error("failed to reclaim stale outbox events", "error", err)
	}
}

func (w *OutboxWorker) ProcessOutbox(ctx context.Context) error {
	_, err := w.ProcessOutboxWithCount(ctx, 1000)
	return err
}

func (w *OutboxWorker) ProcessOutboxWithCount(ctx context.Context, limit int32) (int, error) {
	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var events []db.OutboxEvent

	err := pgx.BeginFunc(opCtx, w.svc.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		var err error
		events, err = q.GetPendingOutboxEventsForUpdate(opCtx, limit)
		if err != nil || len(events) == 0 {
			return err
		}

		ids := make([]int64, len(events))
		for i, ev := range events {
			ids[i] = ev.ID
		}

		_, err = tx.Exec(opCtx, `
			UPDATE outbox_events
			SET status = 'PROCESSING', processing_started_at = NOW()
			WHERE id = ANY($1)`, ids)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil || len(events) == 0 {
		return 0, err
	}

	processedIDs := make([]int64, 0, len(events))
	revertIDs := make([]int64, 0, len(events))
	var batchErrs []error

	for _, ev := range events {
		if err := w.handleOutboxEvent(opCtx, ctx, ev); err != nil {
			slog.Warn("redis outbox processing failed for event, marking for revert", "id", ev.ID, "error", err)
			revertIDs = append(revertIDs, ev.ID)
			batchErrs = append(batchErrs, fmt.Errorf("outbox event %d: %w", ev.ID, err))
			continue
		}
		processedIDs = append(processedIDs, ev.ID)
	}

	if len(processedIDs) > 0 {
		_, err = w.svc.GetPool().Exec(opCtx, "UPDATE outbox_events SET status = 'PROCESSED' WHERE id = ANY($1)", processedIDs)
		if err != nil {
			slog.Error("failed to mark outbox events as processed", "error", err)
			batchErrs = append(batchErrs, fmt.Errorf("mark outbox processed: %w", err))
		}
	}

	if len(revertIDs) > 0 {
		_, err = w.svc.GetPool().Exec(opCtx, `
			UPDATE outbox_events
			SET status = 'PENDING', processing_started_at = NULL
			WHERE id = ANY($1)`, revertIDs)
		if err != nil {
			slog.Error("failed to revert failed outbox events", "error", err)
			batchErrs = append(batchErrs, fmt.Errorf("revert outbox failed: %w", err))
		}
	}

	if len(batchErrs) > 0 {
		return len(processedIDs), errors.Join(batchErrs...)
	}

	return len(processedIDs), nil
}

func (w *OutboxWorker) campaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	var limit, spend int64
	err := w.svc.GetPool().QueryRow(ctx, `
		SELECT budget_limit, current_spend
		FROM campaigns
		WHERE id = $1`, ingestion.ToUUID(campaignID)).Scan(&limit, &spend)
	if err != nil {
		return 0, err
	}
	remaining := limit - spend
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (w *OutboxWorker) setCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
	remaining, err := w.campaignRemainingBudget(ctx, campaignID)
	if err != nil {
		if payloadLimit <= 0 {
			return err
		}
		remaining = payloadLimit
	}
	if remaining <= 0 {
		return nil
	}
	pipe.Set(ctx, fmt.Sprintf("budget:campaign:%s", campaignIDStr), remaining, 0)
	return nil
}

func ToUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

const fraudQuarantineChannel = "fraud:quarantine"
const blacklistUpdateChannel = "blacklist:update"

func (w *OutboxWorker) applyBlacklistPayload(ctx context.Context, p BlacklistPayload, queuedAt time.Time) error {
	if len(w.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	reason := normalizeBlacklistReason(p.Reason)
	key := "blacklist:" + reason
	add := p.Action == "add"
	if p.Action != "add" && p.Action != "remove" {
		return fmt.Errorf("unknown blacklist action: %s", p.Action)
	}
	if err := syncGlobalSetMemberToAllShards(ctx, w.svc.rdbs, key, p.IP, add); err != nil {
		return fmt.Errorf("blacklist sync failed: %w", err)
	}
	if reason == "fraud" && p.Action == "add" {
		_ = publishControlChannelToAllShards(ctx, w.svc.rdbs, fraudQuarantineChannel, p.IP)
	}
	_ = publishControlChannelToAllShards(ctx, w.svc.rdbs, blacklistUpdateChannel, p.IP+":"+reason)
	if !queuedAt.IsZero() {
		lag := time.Since(queuedAt).Seconds()
		if lag >= 0 {
			metrics.BlacklistReplicationLag.Observe(lag)
		}
	}
	return nil
}

func (w *OutboxWorker) syncBrandCreativesToRedis(ctx context.Context, brandIDStr string) error {
	brandID, err := coldpath.ParseUUID(brandIDStr)
	if err != nil {
		return err
	}
	rows, err := db.New(w.svc.GetPool()).ListActiveBrandCreatives(ctx, ToUUID(brandID))
	if err != nil {
		return err
	}
	type creativeEntry struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Weight int32  `json:"weight"`
	}
	entries := make([]creativeEntry, len(rows))
	for i, r := range rows {
		entries[i] = creativeEntry{
			ID:     uuid.UUID(r.ID.Bytes).String(),
			URL:    r.LandingUrl,
			Weight: r.Weight,
		}
	}
	payload, err := coldpath.MarshalJSON(entries)
	if err != nil {
		return err
	}
	if len(w.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client")
	}
	key := "brand:creatives:" + brandIDStr
	for _, rdb := range w.svc.rdbs {
		if err := rdb.Set(ctx, key, payload, 0).Err(); err != nil {
			return err
		}
	}
	return nil
}
