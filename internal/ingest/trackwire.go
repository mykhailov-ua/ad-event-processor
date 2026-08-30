package ingest

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/traceprobe"
	"ad-event-processor/internal/track"
)

type trackProcessor struct {
	filterEngine    *FilterEngine
	registry        domain.CampaignRegistry
	creativeStore   *BrandCreativeStore
	rtbCatalog      *RtbCatalog
	rtbMode         uint8
	settingsWatcher *SettingsWatcher
	ingestGeo       GeoProvider
}

func newTrackProcessor(filterEngine *FilterEngine, registry domain.CampaignRegistry, creativeStore *BrandCreativeStore) trackProcessor {
	if filterEngine != nil {
		filterEngine.SetRegistry(registry)
	}
	return trackProcessor{
		filterEngine:  filterEngine,
		registry:      registry,
		creativeStore: creativeStore,
	}
}

// processTrack runs synchronously on a Tier B pinned worker (LockOSThread). Caller must
// tryAcquireStreamAdmission (TryReserve) before debit and publishAcceptedTrack after accept;
// this file owns filter + landing only, not stream enqueue.
func processTrack(ctx context.Context, p trackProcessor, evt *domain.Event, deviceType []byte) trackOutcome {
	slot := uint32(0)
	if evt != nil {
		slot = uint32(CampaignSlotIndex(evt.CampaignID))
	}
	traceprobe.ProcessTrackEnter(slot)
	out := processTrackInner(ctx, p, evt, deviceType)
	traceprobe.ProcessTrackExit(slot)
	return out
}

func processTrackInner(ctx context.Context, p trackProcessor, evt *domain.Event, deviceType []byte) trackOutcome {
	ensureIngestGeo(p.ingestGeo, evt)
	// RTB_MODE live/shadow may rewrite campaign_id and spend before the main filter chain runs.
	if out, handled := applyRtbAuction(p, evt, deviceType); handled {
		releaseOpenRTB3Scratch(evt)
		return out
	}
	releaseOpenRTB3Scratch(evt)
	// FilterEngine.Check is synchronous on this worker (no go func); includes EVALSHA unless local-quanta full-skip.
	if p.filterEngine != nil {
		if err := p.filterEngine.Check(ctx, evt); err != nil {
			if kind, ok := filter.ClassifyFilterErr(err); ok {
				if kind == filter.FilterRejectFraud {
					// Registry-driven silent accept (202) vs hard 403; not a generic filter reject.
					return fraudTrackOutcome(p.registry, evt)
				}
				return trackOutcome{Status: trackStatusRejected, RejectKind: kind}
			}
			filterEngineFailures.Inc()
			return trackOutcome{Status: trackStatusInternalError}
		}
	}
	// FilterEngine clears FilterDeadlineMono on return; re-arm only for ResolveLandingURL bounded work.
	if evt != nil && p.filterEngine != nil {
		if d := p.filterEngine.Timeout(); d > 0 {
			evt.FilterDeadlineMono = monotonicNano() + d.Nanoseconds()
		}
	}
	landing := ResolveLandingURL(ctx, p.registry, p.creativeStore, evt)
	if evt != nil {
		evt.FilterDeadlineMono = 0
	}
	return trackOutcome{Status: trackStatusAccepted, LandingURL: landing}
}

func fraudTrackOutcome(registry domain.CampaignRegistry, evt *domain.Event) trackOutcome {
	return track.FraudOutcome(registry, evt)
}
