package edge

import (
	"context"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/cilium/ebpf"
	"github.com/redis/go-redis/v9"
)

func recordLRUEvictionBeforeInsert(m *ebpf.Map, mapLabel string, occupied int) {
	if m == nil {
		return
	}
	info, err := m.Info()
	if err != nil || info.MaxEntries == 0 {
		return
	}
	if occupied >= int(info.MaxEntries) {
		metrics.EdgeBlocklistLRUEvictionTotal.WithLabelValues(mapLabel).Inc()
	}
}

func setMapFillRatio(m *ebpf.Map, mapLabel string, occupied int) {
	if m == nil {
		return
	}
	info, err := m.Info()
	if err != nil || info.MaxEntries == 0 {
		return
	}
	metrics.EdgeBlocklistMapFillRatio.WithLabelValues(mapLabel).Set(float64(occupied) / float64(info.MaxEntries))
}

// RecordBlocklistMapMetrics publishes BPF map occupancy for edge-bpf-sync /metrics.
func RecordBlocklistMapMetrics(maps BlocklistMaps, store *BlocklistStore) {
	if store == nil {
		return
	}
	v4Hosts, v6Hosts, v4Prefixes, v6Prefixes := store.Stats()
	setMapFillRatio(maps.V4Host, "blocklist_host_v4", v4Hosts)
	setMapFillRatio(maps.V6Host, "blocklist_host_v6", v6Hosts)
	setMapFillRatio(maps.V4Prefix, "blocklist_v4", v4Prefixes)
	setMapFillRatio(maps.V6Prefix, "blocklist_v6", v6Prefixes)
}

// RecordBlocklistChangelogLagSeconds sets lag from the last consumed changelog score to now.
func RecordBlocklistChangelogLagSeconds(ctx context.Context, redisClient redis.Cmdable, state *BlocklistSyncState) {
	if redisClient == nil || state == nil {
		return
	}
	now := float64(time.Now().Unix())
	lag := now - state.lastScore
	if lag < 0 {
		lag = 0
	}
	zs, err := redisClient.ZRevRangeWithScores(ctx, redisKeyBlacklistChangelogAdd, 0, 0).Result()
	if err == nil && len(zs) > 0 && zs[0].Score > state.lastScore {
		pending := now - zs[0].Score
		if pending > lag {
			lag = pending
		}
	}
	metrics.EdgeBlocklistChangelogLagSeconds.Set(lag)
}
