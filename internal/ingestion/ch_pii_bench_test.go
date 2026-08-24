package ingestion

import (
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"
)

func BenchmarkCHPII_writePathOverhead(b *testing.B) {
	h := piihash.TestHasher()
	events := make([]*domain.Event, 1000)
	for i := range events {
		events[i] = &domain.Event{
			IP:     "198.51.100.42",
			UA:     "Mozilla/5.0",
			UserID: "uid-bench",
		}
	}

	b.Run("baseline_no_hash", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for range events {
			}
		}
	})

	b.Run("pii_hash_batch", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, e := range events {
				_ = hashEventPII(h, e)
			}
		}
	})
}
