package ingestion

import (
	"context"
	"sync"
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
	mu sync.RWMutex
	m  map[string]fraudBlacklistCacheItem
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
		f.shards[i].m = make(map[string]fraudBlacklistCacheItem, 64)
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

	now := time.Now().UnixNano()
	shard.mu.RLock()
	item, ok := shard.m[ip]
	shard.mu.RUnlock()

	if ok && now < item.expiry {
		if item.blacklisted {
			addFraudSignal(evt, FraudReasonL3Blocklist)
		}
		return nil
	}

	redisClient := pickLocalGlobalShard(f.redisShards)
	if redisClient == nil {
		return nil
	}

	onList, err := redisClient.SIsMember(ctx, fraudBlacklistKey, ip).Result()
	if err != nil {
		return nil
	}

	shard.mu.Lock()
	if len(shard.m) >= fraudBlacklistCacheMaxEntriesPerShard {
		fraudBlacklistCachePruneLocked(shard, now)
	}
	shard.m[ip] = fraudBlacklistCacheItem{
		blacklisted: onList,
		expiry:      now + int64(fraudBlacklistCacheTTL),
	}
	shard.mu.Unlock()

	if onList {
		addFraudSignal(evt, FraudReasonL3Blocklist)
	}
	return nil
}

func pickLocalGlobalShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	if len(redisShards) == 0 {
		return nil
	}
	for i := 1; i < len(redisShards); i++ {
		if redisShards[i] != nil {
			return redisShards[i]
		}
	}
	return redisShards[0]
}

func fraudBlacklistCachePruneLocked(shard *fraudBlacklistCacheShard, now int64) {
	for k, v := range shard.m {
		if now >= v.expiry {
			delete(shard.m, k)
		}
	}
	for len(shard.m) >= fraudBlacklistCacheMaxEntriesPerShard {
		for k := range shard.m {
			delete(shard.m, k)
			break
		}
	}
}
