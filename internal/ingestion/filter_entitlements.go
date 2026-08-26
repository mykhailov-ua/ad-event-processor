package ingestion

import (
	"context"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const entitlementsTimezoneCacheShards = 16

type entitlementsTimezoneCache struct {
	shards [entitlementsTimezoneCacheShards]struct {
		mu sync.RWMutex
		m  map[string]*time.Location
	}
}

func (c *entitlementsTimezoneCache) location(timezone string) *time.Location {
	if timezone == "" {
		timezone = "UTC"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(timezone))
	shard := &c.shards[h.Sum32()%entitlementsTimezoneCacheShards]
	shard.mu.RLock()
	loc, ok := shard.m[timezone]
	shard.mu.RUnlock()
	if ok {
		return loc
	}
	loaded, err := time.LoadLocation(timezone)
	if err != nil {
		loaded = time.UTC
	}
	shard.mu.Lock()
	if shard.m == nil {
		shard.m = make(map[string]*time.Location, 4)
	}
	if cached, exists := shard.m[timezone]; exists {
		shard.mu.Unlock()
		return cached
	}
	shard.m[timezone] = loaded
	shard.mu.Unlock()
	return loaded
}

type EntitlementsFilter struct {
	registry          *Registry
	sharder           Sharder
	redisShards       []redis.UniversalClient
	regionCode        uint8
	tzCache           entitlementsTimezoneCache
	cgnatGlobalBypass bool
	mobileCarrierASN  *MobileCarrierASNTable
	asnLookup         ASNLookup
}

func NewEntitlementsFilter(registry *Registry, sharder Sharder, redisShards []redis.UniversalClient) *EntitlementsFilter {
	return &EntitlementsFilter{
		registry:    registry,
		sharder:     sharder,
		redisShards: redisShards,
	}
}

func (f *EntitlementsFilter) SetRegionCode(code uint8) {
	if f != nil {
		f.regionCode = code
	}
}

func (f *EntitlementsFilter) ConfigureCGNAT(globalBypass bool, table *MobileCarrierASNTable, lookup ASNLookup) {
	if f == nil {
		return
	}
	f.cgnatGlobalBypass = globalBypass
	f.mobileCarrierASN = table
	f.asnLookup = lookup
}

func (f *EntitlementsFilter) getRedisShardClient(id uuid.UUID) redis.UniversalClient {
	shard := f.sharder.GetShard(id)
	return f.redisShards[shard]
}

func (f *EntitlementsFilter) Check(ctx context.Context, evt *domain.Event) error {
	campInfo, ok := getCampaignFromEvent(f.registry, evt)
	if !ok {
		return ErrCampaignNotFound
	}
	custID := campInfo.CustomerID

	if evt.Type == "bid" || evt.Type == "rtb" {
		state, depEnt := f.registry.GetLicenseState()
		if !licensing.OpenRTBAllowed(state, depEnt) {
			return ErrLicenseExpired
		}
	}

	ent, ok := f.registry.GetEntitlements(custID)
	if !ok {
		return nil
	}

	if evt.Type == "bid" || evt.Type == "rtb" {
		if !ent.Features.OpenRTBEnabled() {
			return ErrLicenseExpired
		}
	}

	if ent.Limits.MaxRequestsPerDay == 0 {
		return nil
	}

	if cgnatBypassForCampaign(f.cgnatGlobalBypass, f.registry, evt.CampaignID, f.mobileCarrierASN, f.asnLookup, evt.IP, "ingress_rpd") {
		return nil
	}

	timezone := ent.Limits.QuotaResetTimezone
	if timezone == "" {
		timezone = "UTC"
	}

	loc := f.tzCache.location(timezone)

	dateStr := CachedTimeIn(loc).Format("20060102")

	var keyBuf [128]byte
	b := IngressDayKey(keyBuf[:0], f.regionCode, custID, dateStr)
	redisKey := unsafeString(b)

	redisClient := f.getRedisShardClient(custID)
	if redisClient == nil {
		return nil
	}

	pipe := redisClient.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, 28*time.Hour)
	_, execErr := pipe.Exec(ctx)
	if execErr != nil {
		slog.Warn("failed to increment daily quota counter in Redis", "customer_id", custID, "error", execErr)
		return nil
	}

	currentVal := incr.Val()
	if uint64(currentVal) > ent.Limits.MaxRequestsPerDay {
		return ErrDailyQuotaExceeded
	}

	return nil
}
