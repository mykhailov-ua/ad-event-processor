package worker

import (
	"context"
	"fmt"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	OutboxPriSyncBrandCreatives = 1
	OutboxPriCreateCampaign     = 2
	OutboxPriPacing             = 3
	OutboxPriPause              = 4
)

type DeliveryOutboxEntry struct {
	Priority  int
	EventType string
	Payload   []byte
}

type DeliveryOutboxMerge map[uuid.UUID]DeliveryOutboxEntry

func (m DeliveryOutboxMerge) Upsert(campaignID uuid.UUID, priority int, eventType string, payload []byte) {
	if m == nil {
		return
	}
	if existing, ok := m[campaignID]; ok && existing.Priority >= priority {
		return
	}
	m[campaignID] = DeliveryOutboxEntry{
		Priority:  priority,
		EventType: eventType,
		Payload:   payload,
	}
}

func (m DeliveryOutboxMerge) Flush(ctx context.Context, pool pgx.Tx) error {
	if len(m) == 0 {
		return nil
	}
	q := db.New(pool)
	for _, entry := range m {
		if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: entry.EventType,
			Payload:   entry.Payload,
		}); err != nil {
			return fmt.Errorf("flush delivery optimizer outbox %s: %w", entry.EventType, err)
		}
	}
	return nil
}
