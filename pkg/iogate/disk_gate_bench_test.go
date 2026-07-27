package iogate

import (
	"context"
	"testing"
)

func BenchmarkDiskGateAcquire(b *testing.B) {
	g := NewDiskWriteGate(DefaultConfig())
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := g.AcquireAppend(ctx, TierHigh); err != nil {
			b.Fatal(err)
		}
		g.ReleaseAppend(TierHigh)
	}
}

func TestDiskGateAcquire_zeroAlloc(t *testing.T) {
	g := NewDiskWriteGate(DefaultConfig())
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		if err := g.AcquireAppend(ctx, TierHigh); err != nil {
			t.Fatal(err)
		}
		g.ReleaseAppend(TierHigh)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		if err := g.AcquireAppend(ctx, TierHigh); err != nil {
			t.Fatal(err)
		}
		g.ReleaseAppend(TierHigh)
	})
	if allocs != 0 {
		t.Fatalf("AcquireAppend+ReleaseAppend allocs/op = %v, want 0", allocs)
	}
}
