package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type Worker struct {
	host Host
}

func NewWorker(host Host) *Worker {
	return &Worker{host: host}
}

func normalizeBlacklistReason(reason string) string {
	if reason == "" {
		return "manual"
	}
	return reason
}

func (w *Worker) Start(ctx context.Context, interval time.Duration) {
	if err := w.ProcessOutbox(ctx); err != nil {
		slog.Error("outbox startup cold sync failed", "err", err)
	}

	slog.Info("outbox worker starting polling loop", "interval", interval)

	pollBackoff := NewPollBackoff()
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
			if w.host != nil {
				err = w.host.WithPostgresHigh(ctx, func(runCtx context.Context) error {
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
				slog.Error("outbox polling loop iteration failed, retrying in 2s", "err", err)
				pollTimer.Reset(2 * time.Second)
				continue
			}

			pollTimer.Reset(pollBackoff.Next(processed))
		}
	}
}

func (w *Worker) reclaimStaleProcessing(ctx context.Context) {
	_, err := w.host.Pool().Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PENDING', processing_started_at = NULL
		WHERE status = 'PROCESSING'
		 AND processing_started_at IS NOT NULL
		 AND processing_started_at < NOW() - INTERVAL '1 minute'`)
	if err != nil && ctx.Err() == nil && !database.IsShutdownError(err) {
		slog.Error("failed to reclaim stale outbox events", "err", err)
	}
}

func (w *Worker) ProcessOutbox(ctx context.Context) error {
	_, err := w.ProcessOutboxWithCount(ctx, 1000)
	return err
}

func (w *Worker) ProcessOutboxWithCount(ctx context.Context, limit int32) (int, error) {
	opCtx, cancel := workerContext(ctx, WorkerTimeout)
	defer cancel()

	var events []db.OutboxEvent

	err := pgx.BeginFunc(opCtx, w.host.Pool(), func(tx pgx.Tx) error {
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

	blacklistEvents := make([]db.OutboxEvent, 0)
	otherEvents := make([]db.OutboxEvent, 0, len(events))
	for _, ev := range events {
		switch ev.EventType {
		case "UPDATE_BLACKLIST", "ML_BLACKLIST_ADD":
			blacklistEvents = append(blacklistEvents, ev)
		default:
			otherEvents = append(otherEvents, ev)
		}
	}

	if len(blacklistEvents) > 0 {
		if err := w.applyBlacklistOutboxBatch(opCtx, blacklistEvents); err != nil {
			for _, ev := range blacklistEvents {
				revertIDs = append(revertIDs, ev.ID)
				batchErrs = append(batchErrs, fmt.Errorf("outbox event %d: %w", ev.ID, err))
			}
		} else {
			for _, ev := range blacklistEvents {
				processedIDs = append(processedIDs, ev.ID)
			}
		}
	}

	for i, ev := range otherEvents {
		if err := w.handleOutboxEvent(opCtx, ctx, ev); err != nil {
			slog.Warn("redis outbox processing failed for event, halting batch lane", "id", ev.ID, "err", err)
			revertIDs = append(revertIDs, ev.ID)
			batchErrs = append(batchErrs, fmt.Errorf("outbox event %d: %w", ev.ID, err))
			for j := i + 1; j < len(otherEvents); j++ {
				revertIDs = append(revertIDs, otherEvents[j].ID)
			}
			break
		}
		processedIDs = append(processedIDs, ev.ID)
	}

	if len(processedIDs) > 0 {
		_, err = w.host.Pool().Exec(opCtx, "UPDATE outbox_events SET status = 'PROCESSED' WHERE id = ANY($1)", processedIDs)
		if err != nil {
			slog.Error("failed to mark outbox events as processed", "err", err)
			batchErrs = append(batchErrs, fmt.Errorf("mark outbox processed: %w", err))
		}
	}

	if len(revertIDs) > 0 {
		_, err = w.host.Pool().Exec(opCtx, `
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

func (w *Worker) campaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	var limit, spend int64
	err := w.host.Pool().QueryRow(ctx, `
		SELECT budget_limit, current_spend
		FROM campaigns
		WHERE id = $1`, domain.ToUUID(campaignID)).Scan(&limit, &spend)
	if err != nil {
		return 0, err
	}
	remaining := limit - spend
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (w *Worker) setCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
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

const blacklistUpdateChannel = domain.BlacklistUpdateChannel

func (w *Worker) applyBlacklistPayloadsBatch(ctx context.Context, events []db.OutboxEvent) error {
	type reasonBatch struct {
		adds    []string
		removes []string
	}
	byReason := make(map[string]*reasonBatch)
	var maxQueued time.Time

	for _, ev := range events {
		p, err := coldpath.UnmarshalStrict[BlacklistPayload](ev.Payload)
		if err != nil {
			return err
		}
		reason := normalizeBlacklistReason(p.Reason)
		batch, ok := byReason[reason]
		if !ok {
			batch = &reasonBatch{}
			byReason[reason] = batch
		}
		switch p.Action {
		case "add":
			batch.adds = append(batch.adds, p.IP)
		case "remove":
			batch.removes = append(batch.removes, p.IP)
		default:
			return fmt.Errorf("unknown blacklist action: %s", p.Action)
		}
		if ev.CreatedAt.Valid && ev.CreatedAt.Time.After(maxQueued) {
			maxQueued = ev.CreatedAt.Time
		}
	}

	if len(w.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis client available")
	}

	for reason, batch := range byReason {
		key := "blacklist:" + reason
		for i, redisClient := range w.host.RedisShards() {
			if redisClient == nil {
				return fmt.Errorf("redis shard %d is nil", i)
			}
			_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				for _, ip := range batch.removes {
					pipe.SRem(ctx, key, ip)
				}
				if len(batch.adds) > 0 {
					addMembers := make([]interface{}, len(batch.adds))
					for j, ip := range batch.adds {
						addMembers[j] = ip
					}
					pipe.SAdd(ctx, key, addMembers...)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("blacklist batch sync shard %d: %w", i, err)
			}
		}
		if reason == "fraud" && len(batch.adds) > 0 {
			_ = PublishFraudQuarantineBatch(ctx, w.host.RedisShards(), batch.adds)
		}
		for _, ip := range append(batch.adds, batch.removes...) {
			_ = PublishControlChannelToAllShards(ctx, w.host.RedisShards(), blacklistUpdateChannel, ip+":"+reason)
		}
	}

	if !maxQueued.IsZero() {
		lag := time.Since(maxQueued).Seconds()
		if lag >= 0 {
			metrics.BlacklistReplicationLag.Observe(lag)
		}
	}
	return nil
}

func (w *Worker) applyBlacklistPayload(ctx context.Context, p BlacklistPayload, queuedAt time.Time) error {
	if len(w.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis client available")
	}
	reason := normalizeBlacklistReason(p.Reason)
	key := "blacklist:" + reason
	add := p.Action == "add"
	if p.Action != "add" && p.Action != "remove" {
		return fmt.Errorf("unknown blacklist action: %s", p.Action)
	}
	if err := SyncGlobalSetMemberToAllShards(ctx, w.host.RedisShards(), key, p.IP, add); err != nil {
		return fmt.Errorf("blacklist sync failed: %w", err)
	}
	if reason == "fraud" && p.Action == "add" {
		_ = PublishFraudQuarantineBatch(ctx, w.host.RedisShards(), []string{p.IP})
	}
	_ = PublishControlChannelToAllShards(ctx, w.host.RedisShards(), blacklistUpdateChannel, p.IP+":"+reason)
	if !queuedAt.IsZero() {
		lag := time.Since(queuedAt).Seconds()
		if lag >= 0 {
			metrics.BlacklistReplicationLag.Observe(lag)
		}
	}
	return nil
}

func (w *Worker) syncBrandCreativesToRedis(ctx context.Context, brandIDStr string) error {
	brandID, err := coldpath.ParseUUID(brandIDStr)
	if err != nil {
		return err
	}
	rows, err := db.New(w.host.Pool()).ListActiveBrandCreatives(ctx, domain.ToUUID(brandID))
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
	if len(w.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis client")
	}
	key := "brand:creatives:" + brandIDStr
	return SyncKeyToAllShards(ctx, w.host.RedisShards(), key, payload, 0)
}
