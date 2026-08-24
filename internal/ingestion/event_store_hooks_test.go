package ingestion

import (
	"context"
	"errors"
	"testing"

	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

type recordingEventStore struct {
	stored [][]*domain.Event
	err    error
}

func (s *recordingEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	if s.err != nil {
		return s.err
	}
	cp := make([]*domain.Event, len(events))
	copy(cp, events)
	s.stored = append(s.stored, cp)
	return nil
}

func (s *recordingEventStore) Close() error { return nil }

func TestWrapEventStoreAfterBatch_callsAfterSuccess(t *testing.T) {
	inner := &recordingEventStore{}
	var called int
	wrapped := WrapEventStoreAfterBatch(inner, func(ctx context.Context, events []*domain.Event) {
		called++
	})
	evt := &domain.Event{ClickID: "c1", CampaignID: uuid.New(), Type: "conversion"}
	if err := wrapped.StoreBatch(context.Background(), []*domain.Event{evt}); err != nil {
		t.Fatal(err)
	}
	if called != 1 || len(inner.stored) != 1 {
		t.Fatalf("stored=%d called=%d", len(inner.stored), called)
	}
}

func TestWrapEventStoreAfterBatch_skipsAfterFailure(t *testing.T) {
	inner := &recordingEventStore{err: errors.New("fail")}
	called := false
	wrapped := WrapEventStoreAfterBatch(inner, func(ctx context.Context, events []*domain.Event) {
		called = true
	})
	err := wrapped.StoreBatch(context.Background(), []*domain.Event{{Type: "conversion"}})
	if err == nil {
		t.Fatal("want error")
	}
	if called {
		t.Fatal("after hook should not run on store error")
	}
}
