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

func (worker *OutboxWorker) Start(ctx context.Context, interval time.Duration) {
	if err := worker.ProcessOutbox(ctx); err != nil {
		slog.Error("outbox startup cold sync failed", "err", err)
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
			worker.reclaimStaleProcessing(ctx)
			worker.recordOutboxLagMetrics(ctx)
		case <-pollTimer.C:
			var processed int
			var err error
			if worker.svc != nil {
				err = worker.svc.withPgHigh(ctx, func(runCtx context.Context) error {
					var innerErr error
					processed, innerErr = worker.ProcessOutboxWithCount(runCtx, 1000)
					return innerErr
				})
			} else {
				processed, err = worker.ProcessOutboxWithCount(ctx, 1000)
			}
			worker.recordOutboxLagMetrics(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("outbox polling loop iteration failed, retrying in 2s", "err", err)
				pollTimer.Reset(2 * time.Second)
				continue
			}

			pollTimer.Reset(pollBackoff.next(processed))
		}
	}
}

func (worker *OutboxWorker) reclaimStaleProcessing(ctx context.Context) {
	_, err := worker.svc.GetPool().Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PENDING', processing_started_at = NULL
		WHERE status = 'PROCESSING'
		  AND processing_started_at IS NOT NULL
		  AND processing_started_at < NOW() - INTERVAL '1 minute'`)
	if err != nil && ctx.Err() == nil && !database.IsShutdownError(err) {
		slog.Error("failed to reclaim stale outbox events", "err", err)
	}
}

func (worker *OutboxWorker) ProcessOutbox(ctx context.Context) error {
	_, err := worker.ProcessOutboxWithCount(ctx, 1000)
	return err
}

func (worker *OutboxWorker) ProcessOutboxWithCount(ctx context.Context, limit int32) (int, error) {
	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var events []db.OutboxEvent

	err := pgx.BeginFunc(opCtx, worker.svc.GetPool(), func(tx pgx.Tx) error {
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
		if err := worker.handleOutboxEvent(opCtx, ctx, ev); err != nil {
			slog.Warn("redis outbox processing failed for event, marking for revert", "id", ev.ID, "err", err)
			revertIDs = append(revertIDs, ev.ID)
			batchErrs = append(batchErrs, fmt.Errorf("outbox event %d: %w", ev.ID, err))
			continue
		}
		processedIDs = append(processedIDs, ev.ID)
	}

	if len(processedIDs) > 0 {
		_, err = worker.svc.GetPool().Exec(opCtx, "UPDATE outbox_events SET status = 'PROCESSED' WHERE id = ANY($1)", processedIDs)
		if err != nil {
			slog.Error("failed to mark outbox events as processed", "err", err)
			batchErrs = append(batchErrs, fmt.Errorf("mark outbox processed: %w", err))
		}
	}

	if len(revertIDs) > 0 {
		_, err = worker.svc.GetPool().Exec(opCtx, `
			UPDATE outbox_events
			SET status = 'PENDING', processing_started_at = NULL
			WHERE id = ANY($1)`, revertIDs)
		if err != nil {
			slog.Error("failed to revert failed outbox events", "err", err)
			batchErrs = append(batchErrs, fmt.Errorf("revert outbox failed: %w", err))
		}
	}

	if len(batchErrs) > 0 {
		return len(processedIDs), errors.Join(batchErrs...)
	}

	return len(processedIDs), nil
}

func (worker *OutboxWorker) campaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	var limit, spend int64
	err := worker.svc.GetPool().QueryRow(ctx, `
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

func (worker *OutboxWorker) setCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
	remaining, err := worker.campaignRemainingBudget(ctx, campaignID)
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

func (worker *OutboxWorker) applyBlacklistPayload(ctx context.Context, p BlacklistPayload, queuedAt time.Time) error {
	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	reason := normalizeBlacklistReason(p.Reason)
	key := "blacklist:" + reason
	add := p.Action == "add"
	if p.Action != "add" && p.Action != "remove" {
		return fmt.Errorf("unknown blacklist action: %s", p.Action)
	}
	if err := syncGlobalSetMemberToAllShards(ctx, worker.svc.rdbs, key, p.IP, add); err != nil {
		return fmt.Errorf("blacklist sync failed: %w", err)
	}
	if reason == "fraud" && p.Action == "add" {
		_ = publishControlChannelToAllShards(ctx, worker.svc.rdbs, fraudQuarantineChannel, p.IP)
	}
	_ = publishControlChannelToAllShards(ctx, worker.svc.rdbs, blacklistUpdateChannel, p.IP+":"+reason)
	if !queuedAt.IsZero() {
		lag := time.Since(queuedAt).Seconds()
		if lag >= 0 {
			metrics.BlacklistReplicationLag.Observe(lag)
		}
	}
	return nil
}

func (worker *OutboxWorker) syncBrandCreativesToRedis(ctx context.Context, brandIDStr string) error {
	brandID, err := coldpath.ParseUUID(brandIDStr)
	if err != nil {
		return err
	}
	rows, err := db.New(worker.svc.GetPool()).ListActiveBrandCreatives(ctx, ToUUID(brandID))
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
	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client")
	}
	key := "brand:creatives:" + brandIDStr
	for _, rdb := range worker.svc.rdbs {
		if err := rdb.Set(ctx, key, payload, 0).Err(); err != nil {
			return err
		}
	}
	return nil
}
