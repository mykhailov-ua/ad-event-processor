// Redis deny-set read path and full SMEMBERS sync into pinned BPF maps.
//
// Cross-layer (Redis -> BPF map updates):
//
//	Writers (control outbox, admin block IP, RecordAutoBan, fraud worker) mutate Redis SET keys
//	and enqueue ZSET changelog rows via RecordBlacklistChangelog (blocklist_changelog.go).
//
//	SyncBlocklistFromRedis (this file):
//	  SMEMBERS blacklist:manual, active blacklist:auto (TTL-filtered), blacklist:fraud
//	  -> BlocklistStore.ApplyDiff -> LoadPinned maps via caller-held *ebpf.Map handles
//	  -> host keys: blocklist_host_v4/v6 LRU HASH UpdateAny / Delete
//	  -> prefix keys: blocklist_v4/v6 LPM TRIE UpdateAny / Delete
//	  -> IsProtected IPs skipped (aligns with allow-before-deny in edge_filter.c)
//
//	SyncBlocklistIncremental (blocklist_incremental.go):
//	  Every 5 min: full path above (invalidation: bpf_full_resync).
//	  Between: ZRangeByScore on blacklist:changelog:add|remove since lastScore
//	  -> ApplyHostListDelta host-only (invalidation: bpf_changelog_delta).
//	  Prefix/CIDR changes require full resync (incremental is host-IP delta only).
//
//	cmd/edge-bpf-sync opens maps with LoadPinnedBlocklistMap / LoadPinnedBlocklistHostV4Map / ...
//	(blocklist_store.go, blocklist_maps.go, pin_dir.go ResolvePinnedMapPaths).
//
// Parallel L7 path (not this package):
//
//	edge-blacklist-sync.lua: SMEMBERS -> generational _bl_ver + b:{ip}; _bl_count exact on full sync,
//	  incremental adds deduped (no re-count of IPs already at current ver).
//	edge-config.lua: HMGET config:values -> generational _asn_ver (no get_keys scan).
//	perimeter_blacklist_cache.go mirrors full-sync L7 semantics in Go for unit tests (always bumps ver).
//
// Cache invalidation patterns:
//   - bpf_full_resync: ApplyDiff swaps BlocklistStore shadow; explicit map Delete for removals.
//   - bpf_changelog_delta: applyHostRemove -> maps.V4Host.Delete; add -> UpdateAny (+ LRU risk).
//   - bpf_lru_implicit: host map at max_entries evicts in kernel; shadow drifts until full resync.
//   - l7_generational_full / l7_generational_incremental: nginx only; see deploy/nginx/lua/.
package edge

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	redisKeyBlacklistManual  = "blacklist:manual"
	redisKeyBlacklistAuto    = "blacklist:auto"
	redisKeyBlacklistFraud   = "blacklist:fraud"
	redisKeyBlacklistAutoTTL = "blacklist:auto:ttl"
)

type denySetReader interface {
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
}

type autoBanReader interface {
	denySetReader
	ZScore(ctx context.Context, key, member string) *redis.FloatCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd
	ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd
}

func SyncBlocklistFromRedis(ctx context.Context, redisClient denySetReader, maps BlocklistMaps, store *BlocklistStore) (added, removed int, err error) {
	manual, err := redisClient.SMembers(ctx, redisKeyBlacklistManual).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyBlacklistManual, err)
	}
	auto, err := loadAutoBans(ctx, redisClient)
	if err != nil {
		return 0, 0, fmt.Errorf("active %s: %w", redisKeyBlacklistAuto, err)
	}
	fraud, err := redisClient.SMembers(ctx, redisKeyBlacklistFraud).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("smembers %s: %w", redisKeyBlacklistFraud, err)
	}
	return store.ApplyDiff(maps, manual, auto, fraud)
}
