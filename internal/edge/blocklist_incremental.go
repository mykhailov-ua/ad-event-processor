package edge

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/metrics"

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

func loadChangelogDelta(ctx context.Context, redisClient changelogReader, key string, lastScore float64) (members []string, maxScore float64, err error) {
	maxScore = lastScore
	zs, err := redisClient.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
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

func (s *BlocklistStore) ApplyHostListDelta(maps BlocklistMaps, adds, removes []string) (added, removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ip := range adds {
		a, err := s.applyHostAdd(maps, ip)
		if err != nil {
			return added, removed, err
		}
		added += a
	}
	for _, ip := range removes {
		r, err := s.applyHostRemove(maps, ip)
		if err != nil {
			return added, removed, err
		}
		removed += r
	}
	return added, removed, nil
}

func (s *BlocklistStore) applyHostAdd(maps BlocklistMaps, ip string) (int, error) {
	if ip == "" {
		return 0, nil
	}
	if IsProtected(ip) {
		metrics.EdgeBlocklistSkipAllowlistedTotal.Inc()
		return 0, nil
	}
	if v4, ok := ParseHost(ip); ok {
		addr := beToBPFAddr(v4)
		if _, exists := s.hosts[addr]; exists {
			return 0, nil
		}
		if maps.V4Host != nil {
			recordLRUEvictionBeforeInsert(maps.V4Host, "blocklist_host_v4", len(s.hosts))
			if err := maps.V4Host.Update(addr, blockedMarker, ebpf.UpdateAny); err != nil {
				return 0, err
			}
		}
		s.hosts[addr] = struct{}{}
		return 1, nil
	}
	if maps.V6Host == nil {
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
	recordLRUEvictionBeforeInsert(maps.V6Host, "blocklist_host_v6", len(s.v6Hosts))
	if err := maps.V6Host.Update(key.Addr, blockedMarker, ebpf.UpdateAny); err != nil {
		return 0, err
	}
	s.v6Hosts[id] = key
	return 1, nil
}

func (s *BlocklistStore) applyHostRemove(maps BlocklistMaps, ip string) (int, error) {
	if ip == "" {
		return 0, nil
	}
	if v4, ok := ParseHost(ip); ok {
		addr := beToBPFAddr(v4)
		if _, exists := s.hosts[addr]; !exists {
			return 0, nil
		}
		if maps.V4Host != nil {
			if err := maps.V4Host.Delete(addr); err != nil {
				return 0, err
			}
		}
		delete(s.hosts, addr)
		return 1, nil
	}
	if maps.V6Host == nil {
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
	if err := maps.V6Host.Delete(key.Addr); err != nil {
		return 0, err
	}
	delete(s.v6Hosts, id)
	return 1, nil
}

func SyncBlocklistIncremental(
	ctx context.Context,
	redisClient autoBanReader,
	maps BlocklistMaps,
	store *BlocklistStore,
	state *BlocklistSyncState,
) (added, removed int, err error) {
	if store == nil {
		return 0, 0, fmt.Errorf("nil blocklist store")
	}
	now := time.Now()
	if state == nil || state.needsFullSync(now) {
		a, r, err := SyncBlocklistFromRedis(ctx, redisClient, maps, store)
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
	adds, addMax, err := loadChangelogDelta(ctx, redisClient, redisKeyBlacklistChangelogAdd, minScore)
	if err != nil {
		return 0, 0, fmt.Errorf("changelog add: %w", err)
	}
	removes, removeMax, err := loadChangelogDelta(ctx, redisClient, redisKeyBlacklistChangelogRemove, minScore)
	if err != nil {
		return 0, 0, fmt.Errorf("changelog remove: %w", err)
	}

	a, r, err := store.ApplyHostListDelta(maps, adds, removes)
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
