package ingestion

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrSegmentExcluded    = errors.New("segment excluded")
	ErrSegmentNotIncluded = errors.New("segment not included")
)

type segmentMemberCacheKey struct {
	segmentID uuid.UUID
	userHash  [16]byte
}

type segmentMemberCacheItem struct {
	member bool
	expiry int64
}

const (
	segmentMemberCacheShards             = 128
	segmentMemberCacheTTL                = 5 * time.Second
	segmentMemberCacheMaxEntriesPerShard = 2048
)

type segmentMemberCacheShard struct {
	snap atomic.Pointer[segmentMemberShardSnapshot]
}

type segmentMemberShardSnapshot struct {
	entries map[segmentMemberCacheKey]segmentMemberCacheItem
}

type segmentMemberCache struct {
	shards [segmentMemberCacheShards]segmentMemberCacheShard
}

func newSegmentMemberCache() *segmentMemberCache {
	c := &segmentMemberCache{}
	for i := range segmentMemberCacheShards {
		c.shards[i].snap.Store(&segmentMemberShardSnapshot{
			entries: make(map[segmentMemberCacheKey]segmentMemberCacheItem, 64),
		})
	}
	return c
}

func segmentMemberShardStore(shard *segmentMemberCacheShard, key segmentMemberCacheKey, item segmentMemberCacheItem, nowMs int64) {
	for {
		old := shard.snap.Load()
		next := segmentMemberCloneEntries(old, nowMs, key, item)
		newSnap := &segmentMemberShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func segmentMemberShardDelete(shard *segmentMemberCacheShard, key segmentMemberCacheKey) {
	for {
		old := shard.snap.Load()
		if old == nil {
			return
		}
		if _, ok := old.entries[key]; !ok {
			return
		}
		next := make(map[segmentMemberCacheKey]segmentMemberCacheItem, len(old.entries)-1)
		for k, v := range old.entries {
			if k != key {
				next[k] = v
			}
		}
		newSnap := &segmentMemberShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func segmentMemberCloneEntries(old *segmentMemberShardSnapshot, nowMs int64, key segmentMemberCacheKey, item segmentMemberCacheItem) map[segmentMemberCacheKey]segmentMemberCacheItem {
	var oldMap map[segmentMemberCacheKey]segmentMemberCacheItem
	if old != nil {
		oldMap = old.entries
	}
	next := make(map[segmentMemberCacheKey]segmentMemberCacheItem, len(oldMap)+1)
	for k, v := range oldMap {
		if nowMs < v.expiry {
			next[k] = v
		}
	}
	if len(next) >= segmentMemberCacheMaxEntriesPerShard {
		segmentMemberCachePruneMap(next, nowMs)
	}
	next[key] = item
	return next
}

func segmentMemberCachePruneMap(entries map[segmentMemberCacheKey]segmentMemberCacheItem, nowMs int64) {
	for k, v := range entries {
		if nowMs >= v.expiry {
			delete(entries, k)
		}
	}
	for len(entries) >= segmentMemberCacheMaxEntriesPerShard {
		for k := range entries {
			delete(entries, k)
			break
		}
	}
}

func segmentMemberCacheShardIndex(segmentID uuid.UUID, userHash [16]byte) uint32 {
	h := uint32(segmentID[0]) | (uint32(segmentID[1]) << 8)
	h ^= uint32(userHash[0]) | (uint32(userHash[1]) << 8)
	return h % segmentMemberCacheShards
}

func (c *segmentMemberCache) invalidate(segmentID uuid.UUID, userHash [16]byte) {
	if c == nil {
		return
	}
	key := segmentMemberCacheKey{segmentID: segmentID, userHash: userHash}
	shard := &c.shards[segmentMemberCacheShardIndex(segmentID, userHash)]
	segmentMemberShardDelete(shard, key)
}

func (c *segmentMemberCache) memberExists(
	ctx context.Context,
	redisShards []redis.UniversalClient,
	segmentID uuid.UUID,
	userHash [16]byte,
) (bool, error) {
	if c == nil {
		return segmentMemberExists(ctx, redisShards, segmentID, userHash)
	}
	key := segmentMemberCacheKey{segmentID: segmentID, userHash: userHash}
	shardIdx := segmentMemberCacheShardIndex(segmentID, userHash)
	shard := &c.shards[shardIdx]

	nowMs := cachedUnixMilliNow()
	snap := shard.snap.Load()
	if snap != nil {
		if item, ok := snap.entries[key]; ok && nowMs < item.expiry {
			return item.member, nil
		}
	}

	member, err := segmentMemberExists(ctx, redisShards, segmentID, userHash)
	if err != nil {
		return false, err
	}

	segmentMemberShardStore(shard, key, segmentMemberCacheItem{
		member: member,
		expiry: nowMs + segmentMemberCacheTTL.Milliseconds(),
	}, nowMs)
	return member, nil
}

type SegmentFilter struct {
	redisShards []redis.UniversalClient
	registry    domain.CampaignRegistry
	hasher      *piihash.Hasher
	memberCache *segmentMemberCache
}

func NewSegmentFilter(redisShards []redis.UniversalClient, registry domain.CampaignRegistry, hasher *piihash.Hasher) *SegmentFilter {
	return &SegmentFilter{
		redisShards: redisShards,
		registry:    registry,
		hasher:      hasher,
		memberCache: newSegmentMemberCache(),
	}
}

func (f *SegmentFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil || f.registry == nil {
		return nil
	}
	camp, ok := getCampaignFromEvent(f.registry, evt)
	if !ok || camp == nil {
		return nil
	}
	if camp.SegmentIncludeID == uuid.Nil && camp.SegmentExcludeID == uuid.Nil {
		return nil
	}
	userHash, ok := segmentUserHash(f.hasher, evt)
	if !ok {
		if camp.SegmentIncludeID != uuid.Nil {
			return ErrSegmentNotIncluded
		}
		return nil
	}
	if camp.SegmentExcludeID != uuid.Nil {
		member, err := f.memberCache.memberExists(ctx, f.redisShards, camp.SegmentExcludeID, userHash)
		if err != nil {
			return nil
		}
		if member {
			return ErrSegmentExcluded
		}
	}
	if camp.SegmentIncludeID != uuid.Nil {
		member, err := f.memberCache.memberExists(ctx, f.redisShards, camp.SegmentIncludeID, userHash)
		if err != nil {
			return nil
		}
		if !member {
			return ErrSegmentNotIncluded
		}
	}
	return nil
}
