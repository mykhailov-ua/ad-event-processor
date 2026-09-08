package unified

import (
	"context"
	"testing"
	"time"

	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func TestAcceptLocalQuantaFullSkip_Allocs(t *testing.T) {
	shards := []redis.UniversalClient{nil}
	f := NewUnifiedFilter(
		shards,
		filt.NewJumpHashSharder(1),
		nil,
		nil,
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", 1_000_000, 20)
	f.SetLuaFastPathEnabled(true)
	f.SetLocalQuantaMode("live")
	f.clickTryClaimFast = func(string) bool { return true }
	f.streamEnqueueFast = func(int, *domain.Event, *domain.Campaign, int64) bool { return true }

	campID := uuid.New()
	camp := &domain.Campaign{ID: campID, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	evt := &domain.Event{Type: "impression", CampaignID: campID, UserID: "u", IP: "1.2.3.4"}
	const amount = int64(10_000)
	ctx := context.Background()
	var n int
	for i := range 200 {
		n++
		evt.ClickIDBuf[0] = byte('a' + i%26)
		evt.ClickID = filt.UnsafeString(evt.ClickIDBuf[:1])
		_ = f.AcceptLocalQuantaFullSkip(ctx, evt, camp, amount, 0)
	}
	n = 0
	allocs := testing.AllocsPerRun(50, func() {
		n++
		evt.ClickIDBuf[0] = byte('a' + n%26)
		evt.ClickID = filt.UnsafeString(evt.ClickIDBuf[:1])
		_ = f.AcceptLocalQuantaFullSkip(ctx, evt, camp, amount, 0)
	})
	t.Logf("accept noop deps allocs: %v", allocs)
}
