package ingestion

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type getCampaignCallRegistry struct {
	mockRegistry
	calls atomic.Int32
}

func (r *getCampaignCallRegistry) GetCampaign(id uuid.UUID) (*domain.Campaign, bool) {
	r.calls.Add(1)
	return r.mockRegistry.GetCampaign(id)
}

func TestRegistry_GetCampaignWorker_cacheHit(t *testing.T) {
	r := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp, status: db.CampaignStatusTypeACTIVE},
	}})

	got, ok := r.GetCampaignWorker(3, id)
	require.True(t, ok)
	require.Equal(t, camp, got)

	got2, ok2 := r.GetCampaignWorker(3, id)
	require.True(t, ok2)
	require.Equal(t, camp, got2)
}

func TestRegistry_GetCampaignWorker_invalidatesOnReload(t *testing.T) {
	r := NewRegistry(nil)
	id := uuid.New()
	campV1 := &domain.Campaign{ID: id, DailyBudget: 1}
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: campV1},
	}})
	_, ok := r.GetCampaignWorker(1, id)
	require.True(t, ok)

	campV2 := &domain.Campaign{ID: id, DailyBudget: 2}
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: campV2},
	}})
	got, ok := r.GetCampaignWorker(1, id)
	require.True(t, ok)
	require.Equal(t, int64(2), got.DailyBudget)
}

func TestGetCampaignFromEvent_workerCacheHit(t *testing.T) {
	r := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	enrichMockCampaign(camp)
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp},
	}})
	evt := &domain.Event{CampaignID: id, FilterWorkerIdx: 5}

	got, ok := getCampaignFromEvent(r, evt)
	require.True(t, ok)
	require.Equal(t, camp, got)

	got2, ok2 := getCampaignFromEvent(r, evt)
	require.True(t, ok2)
	require.Equal(t, camp, got2)
}

func TestGetCampaignFromEvent_eventCacheSingleRegistryLookup(t *testing.T) {
	campID := uuid.New()
	counter := &getCampaignCallRegistry{}
	configureMockRegistryCampaign(func(c *domain.Campaign) {
		c.ID = campID
	})
	evt := &domain.Event{CampaignID: campID}
	require.False(t, evt.FilterCampResolved)

	for range 6 {
		got, ok := getCampaignFromEvent(counter, evt)
		require.True(t, ok)
		require.NotNil(t, got)
	}
	require.True(t, evt.FilterCampResolved)
	require.Equal(t, int32(1), counter.calls.Load(), "filter chain must reuse per-event campaign cache")
}

func TestScheduleFilter_respectsCachedClock(t *testing.T) {
	SetClockRefreshPaused(true)
	t.Cleanup(func() { SetClockRefreshPaused(false) })

	allowed := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	storeCachedNowUTC()
	cachedNowUTC.Store(&allowed)
	cachedUnixMilli.Store(allowed.UnixMilli())
	cachedUnixMilliAny.Store(allowed.UnixMilli())

	registry := NewRegistry(nil)
	campID := uuid.New()
	custID := uuid.New()
	registry.Add(campID, custID, nil, "", domain.PacingModeAsap, 0, "UTC", 0, 86400, nil)
	snap := registry.campaignMapSnapshot()
	newMap := make(map[uuid.UUID]campaignInfo, len(snap.byID))
	for k, v := range snap.byID {
		newMap[k] = v
	}
	info := newMap[campID]
	info.campaign.DaypartHours = map[int16]struct{}{14: {}}
	newMap[campID] = info
	registry.data.Store(&campaignMapSnapshot{byID: newMap})

	filter := NewScheduleFilter(registry)
	evt := &domain.Event{CampaignID: campID, FilterWorkerIdx: 2, Type: "click"}
	ctx := context.Background()
	require.NoError(t, filter.Check(ctx, evt))

	blocked := time.Date(2026, 3, 15, 3, 0, 0, 0, time.UTC)
	cachedNowUTC.Store(&blocked)
	cachedUnixMilli.Store(blocked.UnixMilli())
	cachedUnixMilliAny.Store(blocked.UnixMilli())
	require.ErrorIs(t, filter.Check(ctx, evt), ErrScheduleBlocked)
}
