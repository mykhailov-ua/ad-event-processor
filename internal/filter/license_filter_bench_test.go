package filter

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
)

func BenchmarkFilterLicense(b *testing.B) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateActive})
	evt := &domain.Event{}
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func TestFilterLicense_zeroAllocs(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateActive})
	evt := &domain.Event{}
	ctx := context.Background()

	allocs := testing.AllocsPerRun(1000, func() {
		_ = f.Check(ctx, evt)
	})
	if allocs > 0 {
		t.Fatalf("LicenseFilter.Check allocated %.1f times per run, want 0", allocs)
	}
}
