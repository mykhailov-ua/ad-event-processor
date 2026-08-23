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
	ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd
}

func changelogMinScoreExclusive(lastScore float64) string {
	return fmt.Sprintf("(%g", lastScore)
}

func loadChangelogDelta(ctx context.Context, rdb changelogReader, key string, lastScore float64) (members []string, maxScore float64, err error) {
	maxScore = lastScore
	zs, err := rdb.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: changelogMinScoreExclusive(lastScore),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, lastScore, err
	}
	members = make([]string, 0, len(zs))
	for _, z := range zs {
		member, ok := z.Member.(string)
		if !ok || member == "" {
			continue
		}
		members = append(members, member)
		if z.Score > maxScore {
			maxScore = z.Score
		}
	}
	return members, maxScore, nil
}

func (s *BlocklistSyncState) needsFullSync(now time.Time) bool {
	if s == nil || s.lastFullSync.IsZero() {
		return true
	}
	return now.Sub(s.lastFullSync) >= blocklistFullSyncInterval
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
	adds, addMax, err := loadChangelogDelta(ctx, rdb, redisKeyBlacklistChangelogAdd, minScore)
	if err != nil {
		return 0, 0, fmt.Errorf("changelog add: %w", err)
	}
	removes, removeMax, err := loadChangelogDelta(ctx, rdb, redisKeyBlacklistChangelogRemove, minScore)
	if err != nil {
		return 0, 0, fmt.Errorf("changelog remove: %w", err)
	}

	a, r, err := store.ApplyHostListDelta(v4Map, v6Map, adds, removes)
	if err != nil {
		return 0, 0, err
	}
	if addMax > state.lastScore {
		state.lastScore = addMax
	}
	if removeMax > state.lastScore {
		state.lastScore = removeMax
	}
	return a, r, nil
}
