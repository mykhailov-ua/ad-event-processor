package unified

import (
	"log/slog"

	filt "ad-event-processor/internal/filter"

	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

func (f *UnifiedFilter) noteLuaEvalDuration(shard int, campaignID uuid.UUID, tier string, startNs int64, sample bool, fast bool) {
	if startNs == 0 {
		return
	}
	elapsedNs := filt.MonotonicNano() - startNs
	if sample {
		sec := float64(elapsedNs) / 1e9
		if fast {
			observeRedisLuaTier(f.luaFastDurationObservers, shard, sec)
		} else {
			observeRedisLua(f.luaDurationObservers, shard, sec)
		}
	}
	if f.filterSlowNs > 0 && elapsedNs > f.filterSlowNs {
		metrics.FilterLuaSlowTotal.Inc()
		slog.Warn("filter lua slow",
			"campaign_id", campaignID,
			"tier", tier,
			"duration_ms", float64(elapsedNs)/1e6,
		)
	}
}
