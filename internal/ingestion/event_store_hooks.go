package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
)

type afterBatchStoreFunc func(ctx context.Context, events []*domain.Event)

type afterBatchEventStore struct {
	inner domain.EventStore
	after afterBatchStoreFunc
}

func WrapEventStoreAfterBatch(inner domain.EventStore, after afterBatchStoreFunc) domain.EventStore {
	if inner == nil || after == nil {
		return inner
	}
	return &afterBatchEventStore{inner: inner, after: after}
}

func (s *afterBatchEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	if err := s.inner.StoreBatch(ctx, events); err != nil {
		return err
	}
	if len(events) > 0 {
		s.after(ctx, events)
	}
	return nil
}

func (s *afterBatchEventStore) Close() error {
	return s.inner.Close()
}
