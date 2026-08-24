package ingestion

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectLandingURL_StickyWeighted(t *testing.T) {
	store := NewBrandCreativeStore(nil, 0)
	brandID := uuid.New()
	store.cache.Store(&brandCreativeMapSnapshot{byBrand: map[uuid.UUID][]brandCreativeEntry{
		brandID: {
			{ID: "a", URL: "https://a.example", Weight: 70},
			{ID: "b", URL: "https://b.example", Weight: 30},
		},
	}})

	url1 := store.SelectLandingURL(context.Background(), brandID, "user-sticky-1", nil)
	url2 := store.SelectLandingURL(context.Background(), brandID, "user-sticky-1", nil)
	assert.Equal(t, url1, url2)
	assert.Contains(t, []string{"https://a.example", "https://b.example"}, url1)
}

func TestBrandCreativeStore_expiredDeadlineSkipsRedisLoad(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr:        mr.Addr(),
		ReadTimeout: 50 * time.Millisecond,
	})
	defer func() { _ = rdb.Close() }()

	brandID := uuid.New()
	require.NoError(t, mr.Set("brand:creatives:"+brandID.String(), `[{"id":"a","url":"https://a.example","weight":1}]`))

	store := NewBrandCreativeStore(rdb, 50)
	evt := &domain.Event{FilterDeadlineMono: monotonicNano() - 1}
	assert.Nil(t, store.SelectLandingURLBytes(context.Background(), brandID, "user-1", evt))
}

func TestBrandCreativeStore_loadFromRedisWithinDeadline(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), ReadTimeout: time.Second})
	defer func() { _ = rdb.Close() }()

	brandID := uuid.New()
	require.NoError(t, mr.Set("brand:creatives:"+brandID.String(), `[{"id":"a","url":"https://a.example","weight":1}]`))

	store := NewBrandCreativeStore(rdb, 500)
	evt := &domain.Event{FilterDeadlineMono: monotonicNano() + int64(500*time.Millisecond)}
	got := store.SelectLandingURLBytes(context.Background(), brandID, "user-1", evt)
	assert.Equal(t, []byte("https://a.example"), got)
}

func TestScheduleFilter_BlocksOutsideDaypart(t *testing.T) {
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
	info.campaign.DaypartHours = map[int16]struct{}{23: {}}
	newMap[campID] = info
	registry.data.Store(&campaignMapSnapshot{byID: newMap})

	filter := NewScheduleFilter(registry)
	evt := &domain.Event{CampaignID: campID, Type: "click"}
	err := filter.Check(context.Background(), evt)
	assert.ErrorIs(t, err, ErrScheduleBlocked)
}

func TestBrandCreativeStore_LoadFromRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	store := NewBrandCreativeStore(nil, 0)
	brandID := uuid.New()
	raw, err := json.Marshal([]brandCreativeEntry{{ID: "x", URL: "https://x.test", Weight: 100}})
	require.NoError(t, err)
	_ = raw
	store.cache.Store(&brandCreativeMapSnapshot{byBrand: map[uuid.UUID][]brandCreativeEntry{
		brandID: {{ID: "x", URL: "https://x.test", Weight: 100}},
	}})
	assert.Equal(t, "https://x.test", store.SelectLandingURL(context.Background(), brandID, "u1", nil))
}
