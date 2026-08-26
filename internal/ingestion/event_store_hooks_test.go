package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
)

type recordingEventStore struct {
	batchCalls int
}

func (s *recordingEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	s.batchCalls++
	return nil
}

func (s *recordingEventStore) Close() error { return nil }

func TestWrapEventStoreBeforeBatch_order(t *testing.T) {
	inner := &recordingEventStore{}
	var beforeCalled bool
	wrapped := WrapEventStoreBeforeBatch(inner, func(ctx context.Context, events []*domain.Event) {
		beforeCalled = true
	})
	if err := wrapped.StoreBatch(context.Background(), []*domain.Event{{Type: "conversion"}}); err != nil {
		t.Fatal(err)
	}
	if !beforeCalled {
		t.Fatal("before hook not called")
	}
	if inner.batchCalls != 1 {
		t.Fatalf("inner calls %d", inner.batchCalls)
	}
}
