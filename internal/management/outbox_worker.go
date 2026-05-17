package management

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/db"
	"github.com/redis/go-redis/v9"
)

// OutboxWorker implements the Transactional Outbox pattern to guarantee eventual consistency between PostgreSQL state modifications and Redis cache invalidation. Processing events asynchronously decouples external cache publication failures from primary database transactions.
type OutboxWorker struct {
	svc *Service
}

func NewOutboxWorker(svc *Service) *OutboxWorker {
	return &OutboxWorker{svc: svc}
}

type CampaignPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit string `json:"budget_limit,omitempty"`
}

type SettingsPayload struct {
	Settings map[string]string `json:"settings"`
}

func (w *OutboxWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.ProcessOutbox(ctx); err != nil {
				if strings.Contains(err.Error(), "closed pool") {
					return
				}
				slog.Error("failed to process outbox events", "error", err)
			}
		}
	}
}

func (w *OutboxWorker) ProcessOutbox(ctx context.Context) error {
	return pgx.BeginFunc(ctx, w.svc.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		events, err := q.GetPendingOutboxEventsForUpdate(ctx, 100)
		if err != nil || len(events) == 0 {
			return err
		}

		for _, ev := range events {
			var rdbErr error
			switch ev.EventType {
			case "CREATE_CAMPAIGN":
				var p CampaignPayload
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					campUUID, _ := uuid.Parse(p.CampaignID)
					rdb := w.svc.getRDB(campUUID)
					if rdb != nil {
						_, rdbErr = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
							pipe.Set(ctx, fmt.Sprintf("budget:campaign:%s", p.CampaignID), p.BudgetLimit, 24*time.Hour)
							channel := w.svc.cfg.CampaignUpdateChannel
							if channel == "" {
								channel = "campaigns:update"
							}
							pipe.Publish(ctx, channel, p.CampaignID)
							return nil
						})
					}
				}
			case "CANCEL_CAMPAIGN":
				var p CampaignPayload
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					campUUID, _ := uuid.Parse(p.CampaignID)
					rdb := w.svc.getRDB(campUUID)
					if rdb != nil {
						_, rdbErr = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
							pipe.Del(ctx, fmt.Sprintf("budget:campaign:%s", p.CampaignID))
							channel := w.svc.cfg.CampaignUpdateChannel
							if channel == "" {
								channel = "campaigns:update"
							}
							pipe.Publish(ctx, channel, p.CampaignID)
							return nil
						})
					}
				}
			case "UPDATE_SETTINGS":
				var p SettingsPayload
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					if len(w.svc.rdbs) > 0 && w.svc.rdbs[0] != nil {
						rdb := w.svc.rdbs[0]
						_, rdbErr = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
							if len(p.Settings) > 0 {
								pipe.HSet(ctx, "config:values", p.Settings)
							}
							pipe.Incr(ctx, "config:version")
							return nil
						})
					}
				}
			}

			if rdbErr == nil {
				_ = q.MarkOutboxEventProcessed(ctx, ev.ID)
			} else {
				slog.Warn("redis outbox processing failed for event", "id", ev.ID, "error", rdbErr)
			}
		}
		return nil
	})
}
