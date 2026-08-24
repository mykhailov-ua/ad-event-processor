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
	for b.Loop() {
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
	for b.Loop() {
		_ = h.HashIP(ip)
	}
}
