package ingestion

import (
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/rtb"
)

func rtbPrefilterReject(watcher *SettingsWatcher, catalog *RtbCatalog, targeting RtbTargetingInput) rtb.NoBidReason {
	if watcher != nil && watcher.Get().EmergencyBreaker {
		return rtb.NoBidBreakerOpen
	}
	if catalog == nil || catalog.registry == nil {
		return rtb.NoBidNone
	}
	if targeting.GeoHash == 0 {
		return rtb.NoBidNone
	}
	shard := catalog.registry.LoadShard(targeting.GeoHash)
	if shard == nil || shard.Count == 0 {
		return rtb.NoBidNoCandidates
	}
	return rtb.NoBidNone
}

func rtbPrebidIVTReject(enabled bool, geo GeoProvider, evt *domain.Event) rtb.NoBidReason {
	if !enabled || evt == nil || geo == nil || evt.IP == "" {
		return rtb.NoBidNone
	}
	anon, err := geo.IsAnonymous(evt.IP)
	if err == nil && anon {
		return rtb.NoBidPrebidIVT
	}
	return rtb.NoBidNone
}
