package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestScheduleFilter_respectsCachedClock(t *testing.T) {
	setClockRefreshPaused(true)
	t.Cleanup(func() { setClockRefreshPaused(false) })

	allowed := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	storeCachedNowUTC()
	cachedNowUTCSetFromUnixMilli(allowed.UnixMilli())
	cachedUnixMilliStore(allowed.UnixMilli())
	cachedUnixMilliAnyStore(allowed.UnixMilli())

	registry := NewRegistry(nil)
	campID := uuid.New()
	custID := uuid.New()
	registry.Add(campID, custID, nil, "", domain.PacingModeAsap, 0, "UTC", 0, 86400, nil)
	registry.PatchCampaignForTest(campID, func(c *domain.Campaign) {
		c.DaypartHours = map[int16]struct{}{14: {}}
	})

	sched := NewScheduleFilter(registry)
	evt := &domain.Event{CampaignID: campID, FilterWorkerIdx: 2, Type: "click"}
	ctx := context.Background()
	require.NoError(t, sched.Check(ctx, evt))

	blocked := time.Date(2026, 3, 15, 3, 0, 0, 0, time.UTC)
	cachedNowUTCSetFromUnixMilli(blocked.UnixMilli())
	cachedUnixMilliStore(blocked.UnixMilli())
	cachedUnixMilliAnyStore(blocked.UnixMilli())
	require.ErrorIs(t, sched.Check(ctx, evt), ErrScheduleBlocked)
}
