package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion/traceprobe"
	"github.com/google/uuid"
)

type trackStatus uint8

const (
	trackStatusAccepted trackStatus = iota
	trackStatusFraudAccepted
	trackStatusRejected
	trackStatusInternalError
)

type trackOutcome struct {
	Status     trackStatus
	RejectKind filterRejectKind
	LandingURL string
}

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
	if out, handled := applyRtbAuction(p, evt, deviceType); handled {
		releaseOpenRTB3Scratch(evt)
		return out
	}
	releaseOpenRTB3Scratch(evt)
	if p.filterEngine != nil {
		if err := p.filterEngine.Check(ctx, evt); err != nil {
			if kind, ok := classifyFilterErr(err); ok {
				if kind == filterRejectFraud {
					return fraudTrackOutcome(p.registry, evt)
				}
				return trackOutcome{Status: trackStatusRejected, RejectKind: kind}
			}
			filterEngineFailures.Inc()
			return trackOutcome{Status: trackStatusInternalError}
		}
	}
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
	if evt != nil && campaignSilentRejectEnabled(registry, evt.CampaignID) {
		evt.SilentRejectEvent = true
		return trackOutcome{Status: trackStatusFraudAccepted, RejectKind: filterRejectFraud}
	}
	return trackOutcome{Status: trackStatusRejected, RejectKind: filterRejectFraudBlocked}
}

func campaignSilentRejectEnabled(registry domain.CampaignRegistry, campaignID uuid.UUID) bool {
	if registry == nil {
		return false
	}
	camp, ok := registry.GetCampaign(campaignID)
	if !ok || camp == nil {
		return false
	}
	return camp.SilentRejectEnabled
}
