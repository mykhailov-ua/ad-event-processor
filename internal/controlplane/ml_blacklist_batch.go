package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"

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

func (worker *OutboxWorker) applyBlacklistOutboxBatch(ctx context.Context, events []db.OutboxEvent) error {
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
		synthetic, err := worker.persistMLBlacklistAdds(ctx, mlEvents)
		if err != nil {
			return err
		}
		redisEvents = append(redisEvents, synthetic...)
	}

	if len(redisEvents) == 0 {
		return nil
	}
	return worker.applyBlacklistPayloadsBatch(ctx, redisEvents)
}

func (worker *OutboxWorker) persistMLBlacklistAdds(ctx context.Context, events []db.OutboxEvent) ([]db.OutboxEvent, error) {
	rows, maxQueued, err := worker.parseMLBlacklistRows(events)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	err = pgx.BeginFunc(ctx, worker.svc.GetPool(), func(tx pgx.Tx) error {
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
		worker.svc.AuditLog(ctx, q, uid, "ML_BLACKLIST_ADD", "system", nil, map[string]any{
			"count":  len(rows),
			"source": "fraud",
		}, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return syntheticBlacklistOutboxEvents(rows, maxQueued)
}

func (worker *OutboxWorker) parseMLBlacklistRows(events []db.OutboxEvent) ([]mlBlacklistPersistRow, time.Time, error) {
	cfg := blacklistTTLFromConfig(worker.svc.cfg)
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

func syntheticBlacklistOutboxEvents(rows []mlBlacklistPersistRow, maxQueued time.Time) ([]db.OutboxEvent, error) {
	out := make([]db.OutboxEvent, 0, len(rows))
	createdAt := pgtype.Timestamptz{}
	if !maxQueued.IsZero() {
		createdAt = pgtype.Timestamptz{Time: maxQueued, Valid: true}
	}

	for _, row := range rows {
		payload, err := coldpath.MarshalOutbox(BlacklistPayload{
			Action: "add",
			IP:     row.ip,
			Reason: row.reason,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal synthetic blacklist payload: %w", err)
		}
		out = append(out, db.OutboxEvent{
			EventType: "UPDATE_BLACKLIST",
			Payload:   payload,
			CreatedAt: createdAt,
		})
	}
	return out, nil
}

func (worker *OutboxWorker) applyMLBlacklistSingle(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.IP == "" {
		return nil
	}

	synthetic, err := worker.persistMLBlacklistAdds(ctx, []db.OutboxEvent{{
		EventType: "ML_BLACKLIST_ADD",
		Payload:   payload,
	}})
	if err != nil {
		return err
	}
	if len(synthetic) == 0 {
		return nil
	}
	return worker.applyBlacklistPayloadsBatch(ctx, synthetic)
}
