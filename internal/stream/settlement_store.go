package stream

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type settlementModeStore struct {
	pg        *PostgresStore
	statsOnly bool
}

func NewSettlementStore(pg *PostgresStore, statsOnly bool) domain.EventStore {
	if pg == nil {
		return nil
	}
	return &settlementModeStore{pg: pg, statsOnly: statsOnly}
}

func (s *settlementModeStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	compacted, dropped := compactSettlementBatch(events)
	if dropped > 0 {
		metrics.SettlementBatchCompactDropped.Add(float64(dropped))
	}
	if s.statsOnly {
		return s.pg.StoreStatsBatch(ctx, compacted)
	}
	return s.pg.StoreBatch(ctx, compacted)
}

func (s *settlementModeStore) Close() error {
	return s.pg.Close()
}
