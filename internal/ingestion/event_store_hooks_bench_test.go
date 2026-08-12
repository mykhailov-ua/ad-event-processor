package ingestion

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

func BenchmarkWrapEventStoreAfterBatch_overhead(b *testing.B) {
	inner := &recordingEventStore{}
	var hookCalls int
	wrapped := WrapEventStoreAfterBatch(inner, func(ctx context.Context, events []*domain.Event) {
		hookCalls++
	})
	evt := &domain.Event{
		ClickID:    "c1",
		CampaignID: uuid.New(),
		Type:       "conversion",
	}
	batch := []*domain.Event{evt}
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = wrapped.StoreBatch(ctx, batch)
	}
}
