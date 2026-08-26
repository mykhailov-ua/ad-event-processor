package ingestion

import (
	"context"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"

	redis "github.com/redis/go-redis/v9"
)

const fraudBlacklistKey = "blacklist:fraud"

type FraudLayer uint8

const (
	FraudLayerNone FraudLayer = iota
	FraudLayerL2Shadow
	FraudLayerL1Reject
)

func decideFraudLayer(acc *fraudAccumulator, tier FraudTier) FraudLayer {
	if acc == nil || acc.count == 0 {
		return FraudLayerNone
	}
	if acc.hasFlags(fraudSignalL3) {
		return FraudLayerL1Reject
	}
	if acc.countFlags(fraudSignalL1High) >= 2 {
		return FraudLayerL1Reject
	}
	if acc.countFlags(fraudSignalL1High) >= 1 ||
		acc.countFlags(fraudSignalL2Weak) >= 1 ||
		tier == FraudTierSuspect ||
		tier == FraudTierIVT ||
		tier == FraudTierBlock {
		return FraudLayerL2Shadow
	}
	return FraudLayerNone
}

func applyFraudLayerDecision(evt *domain.Event, acc *fraudAccumulator, camp *domain.Campaign, boost uint8) (FraudLayer, error) {
	if evt == nil {
		return FraudLayerNone, nil
	}
	evt.ShadowEvent = false

	if acc != nil && boost > 0 && !acc.boostApplied {
		sum := acc.score + uint32(boost)
		if sum > 100 {
			sum = 100
		}
		acc.score = sum
		acc.boostApplied = true
	}

	tier := applyFraudAccumulatorForCampaign(evt, acc, camp)
	if acc == nil || acc.count == 0 {
		return FraudLayerNone, nil
	}

	layer := decideFraudLayer(acc, tier)
	recordFraudMetrics(acc, tier, layer)

	switch layer {
	case FraudLayerL1Reject:
		return FraudLayerL1Reject, ErrFraudDetected
	case FraudLayerL2Shadow:
		evt.ShadowEvent = true
		return FraudLayerL2Shadow, nil
	default:
		return FraudLayerNone, nil
	}
}

type fraudBlacklistCacheItem struct {
	blacklisted bool
	expiry      int64
}

const fraudBlacklistCacheTTL = 5 * time.Second

const fraudBlacklistCacheShards = 128

const fraudBlacklistCacheMaxEntriesPerShard = 2048

type fraudBlacklistCacheShard struct {
	snap atomic.Pointer[fraudBlacklistShardSnapshot]
}

type fraudBlacklistShardSnapshot struct {
	entries map[string]fraudBlacklistCacheItem
}

type FraudBlacklistFilter struct {
	redisShards []redis.UniversalClient
	shards      [fraudBlacklistCacheShards]fraudBlacklistCacheShard
}

func NewFraudBlacklistFilter(redisShards []redis.UniversalClient) *FraudBlacklistFilter {
	if len(redisShards) == 0 {
		return nil
	}
	f := &FraudBlacklistFilter{redisShards: redisShards}
	for i := range fraudBlacklistCacheShards {
		f.shards[i].snap.Store(&fraudBlacklistShardSnapshot{
			entries: make(map[string]fraudBlacklistCacheItem, 64),
		})
	}
	return f
}

func fraudBlacklistShardIndex(ip string) uint32 {
	if len(ip) == 0 {
		return 0
	}
	h := uint32(ip[0])
	if len(ip) > 1 {
		h |= uint32(ip[1]) << 8
	}
	return h % fraudBlacklistCacheShards
}

func (f *FraudBlacklistFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || evt.IP == "" {
		return nil
	}

	ip := evt.IP
	shardIdx := fraudBlacklistShardIndex(ip)
	shard := &f.shards[shardIdx]

	nowMs := cachedUnixMilliNow()
	snap := shard.snap.Load()
	if snap != nil {
		if item, ok := snap.entries[ip]; ok && nowMs < item.expiry {
			if item.blacklisted {
				addFraudSignal(evt, FraudReasonL3Blocklist)
			}
			return nil
		}
	}

	redisClient := pickGlobalReadShardForIP(f.redisShards, ip)
	if redisClient == nil {
		return nil
	}

	onList, err := redisClient.SIsMember(ctx, fraudBlacklistKey, ip).Result()
	if err != nil {
		return nil
	}

	fraudBlacklistShardStore(shard, ip, fraudBlacklistCacheItem{
		blacklisted: onList,
		expiry:      nowMs + fraudBlacklistCacheTTL.Milliseconds(),
	}, nowMs)

	if onList {
		addFraudSignal(evt, FraudReasonL3Blocklist)
	}
	return nil
}

func fraudBlacklistShardStore(shard *fraudBlacklistCacheShard, ip string, item fraudBlacklistCacheItem, nowMs int64) {
	for {
		old := shard.snap.Load()
		next := fraudBlacklistCloneEntries(old, nowMs, ip, item)
		newSnap := &fraudBlacklistShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func fraudBlacklistShardDeleteIP(shard *fraudBlacklistCacheShard, ip string) {
	for {
		old := shard.snap.Load()
		if old == nil {
			return
		}
		if _, ok := old.entries[ip]; !ok {
			return
		}
		next := make(map[string]fraudBlacklistCacheItem, len(old.entries)-1)
		for k, v := range old.entries {
			if k != ip {
				next[k] = v
			}
		}
		newSnap := &fraudBlacklistShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func fraudBlacklistCloneEntries(old *fraudBlacklistShardSnapshot, nowMs int64, ip string, item fraudBlacklistCacheItem) map[string]fraudBlacklistCacheItem {
	var oldMap map[string]fraudBlacklistCacheItem
	if old != nil {
		oldMap = old.entries
	}
	next := make(map[string]fraudBlacklistCacheItem, len(oldMap)+1)
	for k, v := range oldMap {
		if nowMs < v.expiry {
			next[k] = v
		}
	}
	if len(next) >= fraudBlacklistCacheMaxEntriesPerShard {
		fraudBlacklistCachePruneMap(next, nowMs)
	}
	next[ip] = item
	return next
}

func fraudBlacklistCachePruneMap(entries map[string]fraudBlacklistCacheItem, now int64) {
	for k, v := range entries {
		if now >= v.expiry {
			delete(entries, k)
		}
	}
	for len(entries) >= fraudBlacklistCacheMaxEntriesPerShard {
		for k := range entries {
			delete(entries, k)
			break
		}
	}
}
