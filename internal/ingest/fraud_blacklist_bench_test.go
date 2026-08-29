//go:build !race

package ingest

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
