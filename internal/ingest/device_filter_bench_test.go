//go:build !race

package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func BenchmarkDeviceFilter_Check_clean(b *testing.B) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	sw.StoreDynamicConfigForTest(&DynamicConfig{TLSHashBlocklist: "blockedhash"})
	f := NewDeviceFilter(sw)

	evt := &domain.Event{
		CampaignID: uuid.New(),
		UA:         "Mozilla/5.0 Chrome/120",
		SecCHUA:    `"Google Chrome";v="120"`,
		TLSHash:    "cleanhash",
	}
	ctx := context.Background()
	for range 1000 {
		_ = f.Check(ctx, evt)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}
