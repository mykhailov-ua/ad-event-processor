package stream

import (
	"context"

	"ad-event-processor/internal/domain"
)

// afterBatchStoreFunc runs only after the wrapped EventStore.StoreBatch succeeds.
// Used on processor settlement for post-CH side effects (e.g. conversion postback outbox).
type afterBatchStoreFunc func(ctx context.Context, events []*domain.Event)

type afterBatchEventStore struct {
	inner domain.EventStore
	after afterBatchStoreFunc
}

// WrapEventStoreAfterBatch decorates inner with a post-success hook. Nil inner or after returns inner unchanged.
// In cmd/processor, afterStored (postback outbox) wraps the PG settlement store after CH-first wiring.
//
// Invariant: after is not called when inner.StoreBatch returns an error or when events is empty.
//
// Verify:
// go test ./internal/stream/ -short -run TestWrapEventStoreBeforeBatch_order -count=1
// go test ./internal/stream/ -bench=BenchmarkWrapEventStoreAfterBatch_overhead -benchmem -count=1
func WrapEventStoreAfterBatch(inner domain.EventStore, after afterBatchStoreFunc) domain.EventStore {
	if inner == nil || after == nil {
		return inner
	}
	return &afterBatchEventStore{inner: inner, after: after}
}

// beforeBatchStoreFunc mutates or filters the batch before the inner store persists it.
type beforeBatchStoreFunc func(ctx context.Context, events []*domain.Event)

type beforeBatchEventStore struct {
	inner  domain.EventStore
	before beforeBatchStoreFunc
}

// WrapEventStoreBeforeBatch decorates inner with a pre-store hook. Nil inner or before returns inner unchanged.
//
// Registration order (cmd/processor): AfterBatch(postback) is applied first, then BeforeBatch(conversion
// smart reject) wraps the outside. StoreBatch call order: before hook -> inner store (PG stats) ->
// after hook. ClickHouseStore runs conversion reject inline in StoreBatch before insert, not via this wrapper.
//
// Invariant: before runs on the same []*domain.Event slice passed to inner.StoreBatch; skip when len(events)==0.
//
// Verify:
// go test ./internal/stream/ -short -run TestWrapEventStoreBeforeBatch_order -count=1
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
