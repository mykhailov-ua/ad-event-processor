package piihash_test

import (
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"
)

func BenchmarkPIIHash_batch1000(b *testing.B) {
	h := piihash.TestHasher()
	events := make([]*domain.Event, 1000)
	for i := range events {
		events[i] = &domain.Event{
			IP:     "203.0.113.42",
			UA:     "Mozilla/5.0 (bench)",
			UserID: "user-bench",
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, e := range events {
			_ = h.HashIP(e.IP)
			_ = h.HashUA(e.UA)
			_ = h.HashUserID(e.UserID)
		}
	}
}

func BenchmarkPIIHash_singleIP(b *testing.B) {
	h := piihash.TestHasher()
	ip := "203.0.113.42"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.HashIP(ip)
	}
}
