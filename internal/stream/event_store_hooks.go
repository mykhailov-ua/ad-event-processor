package stream

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

type beforeBatchStoreFunc func(ctx context.Context, events []*domain.Event)

type beforeBatchEventStore struct {
	inner  domain.EventStore
	before beforeBatchStoreFunc
}

func WrapEventStoreBeforeBatch(inner domain.EventStore, before beforeBatchStoreFunc) domain.EventStore {
	if inner == nil || before == nil {
		return inner
	}
	return &beforeBatchEventStore{inner: inner, before: before}
}

func (s *beforeBatchEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) > 0 && s.before != nil {
		s.before(ctx, events)
	}
	return s.inner.StoreBatch(ctx, events)
}

func (s *beforeBatchEventStore) Close() error {
	return s.inner.Close()
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
