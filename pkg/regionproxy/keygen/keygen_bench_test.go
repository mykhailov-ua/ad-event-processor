package keygen

import (
	"testing"

	"espx/pkg/dedupkey"
	"espx/pkg/regionproxy/wal"
)

func BenchmarkFactorUDerive(b *testing.B) {
	payload := []byte(`{"campaign":"bench","click":"x"}`)
	var buf [wal.MaxPayloadSize + 64]byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canon := dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], uint64(i), payload)
		_ = dedupkey.FactorU(canon)
	}
}
