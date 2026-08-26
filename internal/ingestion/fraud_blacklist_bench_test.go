//go:build !race

package ingestion

import (
	"testing"
)

func BenchmarkFraudBlacklistFilter_hit(b *testing.B) {
	f, evt, ctx := setupFraudBlacklistBench(b, true)
	b.ReportAllocs()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}
