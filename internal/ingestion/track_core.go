package ingestion

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ingestion/traceprobe"
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

func processTrack(p trackProcessor, evt *domain.Event, deviceType []byte) trackOutcome {
	slot := uint32(0)
	if evt != nil {
		slot = uint32(CampaignSlotIndex(evt.CampaignID))
	}
	traceprobe.ProcessTrackEnter(slot)
	out := processTrackInner(p, evt, deviceType)
	traceprobe.ProcessTrackExit(slot)
	return out
}

func processTrackInner(p trackProcessor, evt *domain.Event, deviceType []byte) trackOutcome {
	ensureIngestGeo(p.ingestGeo, evt)
	if out, handled := applyRtbAuction(p, evt, deviceType); handled {
		releaseOpenRTB3Scratch(evt)
		return out
	}
	releaseOpenRTB3Scratch(evt)
	if p.filterEngine != nil {
		if err := p.filterEngine.Check(context.Background(), evt); err != nil {
			if kind, ok := classifyFilterErr(err); ok {
				if kind == filterRejectFraud {
					return trackOutcome{Status: trackStatusFraudAccepted, RejectKind: kind}
				}
				return trackOutcome{Status: trackStatusRejected, RejectKind: kind}
			}
			filterEngineFailures.Inc()
			return trackOutcome{Status: trackStatusInternalError}
		}
	}
	landing := ResolveLandingURL(p.registry, p.creativeStore, evt)
	return trackOutcome{Status: trackStatusAccepted, LandingURL: landing}
}
