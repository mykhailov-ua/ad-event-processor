package keygen

import (
	"testing"

	"ad-event-processor/pkg/dedupkey"
	"ad-event-processor/pkg/regionproxy/wal"
)

func BenchmarkFactorUDerive(b *testing.B) {
	payload := []byte(`{"campaign":"bench","click":"x"}`)
	var buf [wal.MaxPayloadSize + 64]byte
	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		canon := dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], uint64(benchN), payload)
		_ = dedupkey.FactorU(canon)
		benchN++
	}
}
