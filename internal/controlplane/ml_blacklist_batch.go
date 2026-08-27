package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type mlBlacklistPersistRow struct {
	ip        string
	reason    string
	expiresAt pgtype.Timestamptz
	ttl       pgtype.Int4
}

func (w *OutboxWorker) applyBlacklistOutboxBatch(ctx context.Context, events []db.OutboxEvent) error {
	var redisEvents []db.OutboxEvent
	var mlEvents []db.OutboxEvent
	for _, ev := range events {
		if ev.EventType == "ML_BLACKLIST_ADD" {
			mlEvents = append(mlEvents, ev)
		} else {
			redisEvents = append(redisEvents, ev)
		}
	}

	if len(mlEvents) > 0 {
		if err := w.persistMLBlacklistAdds(ctx, mlEvents); err != nil {
			return err
		}
	}

	if len(redisEvents) == 0 {
		return nil
	}
	return w.applyBlacklistPayloadsBatch(ctx, redisEvents)
}

func (w *OutboxWorker) persistMLBlacklistAdds(ctx context.Context, events []db.OutboxEvent) error {
	rows, maxQueued, err := w.parseMLBlacklistRows(events)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	err = pgx.BeginFunc(ctx, w.svc.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		for _, row := range rows {
			if _, err := q.CreateBlacklistIP(ctx, db.CreateBlacklistIPParams{
				Ip:        row.ip,
				Reason:    row.reason,
				ExpiresAt: row.expiresAt,
			}); err != nil {
				return err
			}
			if _, err := q.CreateEdgeBlockAudit(ctx, db.CreateEdgeBlockAuditParams{
				Ip:       row.ip,
				ReasonID: row.reason,
				Ttl:      row.ttl,
				Source:   "fraud",
			}); err != nil {
				return err
			}
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		w.svc.AuditLog(ctx, q, uid, "ML_BLACKLIST_ADD", "system", nil, map[string]any{
			"count":  len(rows),
			"source": "fraud",
		}, nil)
		return nil
	})
	if err != nil {
		return err
	}

	return w.applyMLBlacklistRedisFastLane(ctx, rows, maxQueued)
}

func (w *OutboxWorker) applyMLBlacklistRedisFastLane(ctx context.Context, rows []mlBlacklistPersistRow, maxQueued time.Time) error {
	if len(rows) == 0 {
		return nil
	}
	if len(w.svc.redisShards) == 0 {
		return fmt.Errorf("no redis client available")
	}

	ips := make([]string, len(rows))
	for i, row := range rows {
		ips[i] = row.ip
	}

	key := "blacklist:fraud"
	addMembers := make([]interface{}, len(ips))
	for i, ip := range ips {
		addMembers[i] = ip
	}

	for i, redisClient := range w.svc.redisShards {
		if redisClient == nil {
			return fmt.Errorf("redis shard %d is nil", i)
		}
		if err := redisClient.SAdd(ctx, key, addMembers...).Err(); err != nil {
			return fmt.Errorf("ml blacklist fast lane shard %d: %w", i, err)
		}
	}

	if err := publishFraudQuarantineBatch(ctx, w.svc.redisShards, ips); err != nil {
		return fmt.Errorf("ml blacklist quarantine fast lane: %w", err)
	}
	for _, ip := range ips {
		_ = publishControlChannelToAllShards(ctx, w.svc.redisShards, blacklistUpdateChannel, ip+":fraud")
	}

	if !maxQueued.IsZero() {
		lag := time.Since(maxQueued).Seconds()
		if lag >= 0 {
			metrics.BlacklistReplicationLag.Observe(lag)
		}
	}
	return nil
}

func (w *OutboxWorker) parseMLBlacklistRows(events []db.OutboxEvent) ([]mlBlacklistPersistRow, time.Time, error) {
	cfg := blacklistTTLFromConfig(w.svc.cfg)
	rows := make([]mlBlacklistPersistRow, 0, len(events))
	var maxQueued time.Time

	for _, ev := range events {
		p, err := coldpath.UnmarshalStrict[FraudThreatPayload](ev.Payload)
		if err != nil {
			return nil, time.Time{}, err
		}
		if p.IP == "" {
			continue
		}
		if edge.IsProtected(p.IP) {
			slog.Warn("ml blacklist skip protected ip", "ip", p.IP)
			continue
		}

		reason := normalizeBlacklistReason("fraud")
		ttlSec := p.TTLSeconds
		expiresAt := resolveBlacklistExpiry(reason, &ttlSec, cfg)

		var ttlVal pgtype.Int4
		if expiresAt.Valid {
			diff := expiresAt.Time.Sub(time.Now().UTC())
			if diff > 0 {
				ttlVal = pgtype.Int4{Int32: int32(diff.Seconds()), Valid: true}
			}
		}

		rows = append(rows, mlBlacklistPersistRow{
			ip:        p.IP,
			reason:    reason,
			expiresAt: expiresAt,
			ttl:       ttlVal,
		})

		if ev.CreatedAt.Valid && ev.CreatedAt.Time.After(maxQueued) {
			maxQueued = ev.CreatedAt.Time
		}
	}

	return rows, maxQueued, nil
}

func (w *OutboxWorker) applyMLBlacklistSingle(ctx context.Context, payload []byte) error {
	return w.persistMLBlacklistAdds(ctx, []db.OutboxEvent{{
		EventType: "ML_BLACKLIST_ADD",
		Payload:   payload,
	}})
}
