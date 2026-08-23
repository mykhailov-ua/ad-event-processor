package edge

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/cilium/ebpf"
	"github.com/redis/go-redis/v9"
)

const blocklistFullSyncInterval = 5 * time.Minute

type BlocklistSyncState struct {
	lastFullSync time.Time
	lastScore    float64
}

type changelogReader interface {
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd
}

func (s *BlocklistSyncState) needsFullSync(now time.Time) bool {
	if s == nil || s.lastFullSync.IsZero() {
		return true
	}
	return now.Sub(s.lastFullSync) >= blocklistFullSyncInterval
}

func loadChangelogSince(ctx context.Context, rdb changelogReader, key string, minScore float64) ([]string, error) {
	cmd := rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", minScore),
		Max: "+inf",
	})
	members, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (s *BlocklistStore) ApplyHostListDelta(v4Map, v6Map *ebpf.Map, adds, removes []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ip := range adds {
		a, err := s.applyHostAdd(v4Map, v6Map, ip)
		if err != nil {
			return added, removed, err
		}
		added += a
	}
	for _, ip := range removes {
		r, err := s.applyHostRemove(v4Map, v6Map, ip)
		if err != nil {
			return added, removed, err
		}
		removed += r
	}
	return added, removed, nil
}

func (s *BlocklistStore) applyHostAdd(v4Map, v6Map *ebpf.Map, ip string) (int, error) {
	if ip == "" {
		return 0, nil
	}
	if IsProtected(ip) {
		metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
		return 0, nil
	}
	if v4, ok := ParseHost(ip); ok {
		if _, exists := s.hosts[v4]; exists {
			return 0, nil
		}
		if v4Map != nil {
			if err := v4Map.Update(KeyFromIP(v4), blockedMarker, ebpf.UpdateAny); err != nil {
				return 0, err
			}
		}
		s.hosts[v4] = struct{}{}
		return 1, nil
	}
	if v6Map == nil {
		return 0, nil
	}
	key, ok := ParseIPv6Host(ip)
	if !ok {
		return 0, nil
	}
	id := key.StoreKey()
	if _, exists := s.v6Hosts[id]; exists {
		return 0, nil
	}
	if err := v6Map.Update(key, blockedMarker, ebpf.UpdateAny); err != nil {
		return 0, err
	}
	s.v6Hosts[id] = key
	return 1, nil
}

func (s *BlocklistStore) applyHostRemove(v4Map, v6Map *ebpf.Map, ip string) (int, error) {
	if ip == "" {
		return 0, nil
	}
	if v4, ok := ParseHost(ip); ok {
		if _, exists := s.hosts[v4]; !exists {
			return 0, nil
		}
		if v4Map != nil {
			if err := v4Map.Delete(KeyFromIP(v4)); err != nil {
				return 0, err
			}
		}
		delete(s.hosts, v4)
		return 1, nil
	}
	if v6Map == nil {
		return 0, nil
	}
	key, ok := ParseIPv6Host(ip)
	if !ok {
		return 0, nil
	}
	id := key.StoreKey()
	if _, exists := s.v6Hosts[id]; !exists {
		return 0, nil
	}
	if err := v6Map.Delete(key); err != nil {
		return 0, err
	}
	delete(s.v6Hosts, id)
	return 1, nil
}

// SyncBlocklistIncremental applies changelog deltas between periodic full SMEMBERS refreshes.
func SyncBlocklistIncremental(
	ctx context.Context,
	rdb autoBanReader,
	v4Map, v6Map *ebpf.Map,
	store *BlocklistStore,
	state *BlocklistSyncState,
) (added, removed int, err error) {
	if store == nil {
		return 0, 0, fmt.Errorf("nil blocklist store")
	}
	now := time.Now()
	if state == nil || state.needsFullSync(now) {
		a, r, err := SyncBlocklistFromRedis(ctx, rdb, v4Map, v6Map, store)
		if err != nil {
			return 0, 0, err
		}
		if state != nil {
			state.lastFullSync = now
			state.lastScore = float64(now.Unix())
		}
		return a, r, nil
	}

	minScore := state.lastScore
	adds, err := loadChangelogSince(ctx, rdb, redisKeyBlacklistChangelogAdd, minScore)
	if err != nil {
		return 0, 0, fmt.Errorf("changelog add: %w", err)
	}
	removes, err := loadChangelogSince(ctx, rdb, redisKeyBlacklistChangelogRemove, minScore)
	if err != nil {
		return 0, 0, fmt.Errorf("changelog remove: %w", err)
	}

	a, r, err := store.ApplyHostListDelta(v4Map, v6Map, adds, removes)
	if err != nil {
		return 0, 0, err
	}
	state.lastScore = float64(now.Unix())
	return a, r, nil
}
