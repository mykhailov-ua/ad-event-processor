package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/redis/go-redis/v9"
)

type DealFloorCache struct {
	redisClient redis.UniversalClient
	snap        atomic.Pointer[map[string]int64]
}

func NewDealFloorCache(redisClient redis.UniversalClient) *DealFloorCache {
	c := &DealFloorCache{redisClient: redisClient}
	empty := make(map[string]int64)
	c.snap.Store(&empty)
	return c
}

func (c *DealFloorCache) Get(dealID string) (int64, bool) {
	if dealID == "" {
		return 0, false
	}
	ptr := c.snap.Load()
	if ptr == nil {
		return 0, false
	}
	v, ok := (*ptr)[dealID]
	return v, ok
}

func (c *DealFloorCache) Refresh(ctx context.Context, dealIDs []string) {
	if c == nil || c.redisClient == nil || len(dealIDs) == 0 {
		return
	}
	keys := make([]string, len(dealIDs))
	for i, id := range dealIDs {
		keys[i] = domain.RtbFloorRedisKeyPrefix + id
	}
	vals, err := c.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		slog.Warn("deal floor cache refresh failed", "error", err)
		return
	}
	next := make(map[string]int64, len(dealIDs))
	for i, raw := range vals {
		if raw == nil {
			continue
		}
		s, ok := raw.(string)
		if !ok || s == "" {
			continue
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v <= 0 {
			continue
		}
		next[dealIDs[i]] = v
	}
	c.snap.Store(&next)
}

func StartDealFloorRefresh(ctx context.Context, cache *DealFloorCache, catalog *RtbCatalog, interval time.Duration) {
	if cache == nil || catalog == nil || interval <= 0 {
		return
	}
	refresh := func() {
		deals := catalog.AllDeals()
		if len(deals) == 0 {
			return
		}
		ids := make([]string, len(deals))
		for i, d := range deals {
			ids[i] = d.DealID
		}
		rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cache.Refresh(rctx, ids)
		cancel()
	}
	refresh()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
}

func EffectiveDealFloor(catalog *RtbCatalog, floors *DealFloorCache, dealID string, publisherFloor int64) int64 {
	return EffectiveDealFloorBytes(catalog, floors, []byte(dealID), publisherFloor)
}

func EffectiveDealFloorBytes(catalog *RtbCatalog, floors *DealFloorCache, dealID []byte, publisherFloor int64) int64 {
	floor := publisherFloor
	if len(dealID) == 0 {
		return floor
	}
	if catalog != nil {
		if deal, ok := catalog.LookupDealBytes(dealID); ok && deal.FloorMicro > floor {
			floor = deal.FloorMicro
		}
	}
	if floors != nil {
		key := UnsafeString(dealID)
		if optimized, ok := floors.Get(key); ok && optimized > floor {
			floor = optimized
		}
	}
	return floor
}
