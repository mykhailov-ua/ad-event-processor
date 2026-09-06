package ingest

import (
	"context"
	"net/http"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

// TrackPublishDeps is the canonical reserve -> debit (caller) -> publish -> rollback surface for /track sinks.
type TrackPublishDeps struct {
	Cfg             *config.Config
	Sharder         Sharder
	StreamProducers []*StreamProducer
	BrokerProducers *BrokerProducerSet
	FilterEngine    *FilterEngine
	Registry        domain.CampaignRegistry
}

func (h *AdsPacketHandler) trackPublishDeps() TrackPublishDeps {
	if h == nil {
		return TrackPublishDeps{}
	}
	return TrackPublishDeps{
		Cfg:             h.cfg,
		Sharder:         h.sharder,
		StreamProducers: h.streamProducers,
		BrokerProducers: h.brokerProducers,
		FilterEngine:    h.filterEngine,
		Registry:        h.registry,
	}
}

func (d TrackPublishDeps) Reserve(campaignID uuid.UUID) (streamAdmissionLease, filter.FilterRejectKind, bool) {
	return tryAcquireStreamAdmissionForFilter(
		d.Cfg,
		d.Sharder,
		d.StreamProducers,
		d.BrokerProducers,
		campaignID,
		d.FilterEngine,
	)
}

func (d TrackPublishDeps) PublishAccepted(evt *domain.Event, lease *streamAdmissionLease) bool {
	return publishAcceptedTrackIngress(
		d.Sharder,
		d.StreamProducers,
		d.BrokerProducers,
		d.FilterEngine,
		evt,
		lease,
	)
}

// PublishAcceptedOrRollback enqueues after filter accept. On failure rolls back Lua debit when FilterEngine is set.
//
// Verify:
// go test ./internal/ingest/ -short -run TestPublishAcceptedOrRollback_holdout -count=1
func (d TrackPublishDeps) PublishAcceptedOrRollback(ctx context.Context, evt *domain.Event, lease *streamAdmissionLease) bool {
	if d.PublishAccepted(evt, lease) {
		return true
	}
	if d.FilterEngine != nil && evt != nil {
		d.FilterEngine.RollbackDebit(ctx, evt, d.Registry)
	}
	return false
}

func (h *AdsPacketHandler) publishAcceptedOrRollback(ctx context.Context, evt *domain.Event, lease *streamAdmissionLease) bool {
	return h.trackPublishDeps().PublishAcceptedOrRollback(ctx, evt, lease)
}

// streamAdmissionGuard pairs TryReserve with Release on filter reject paths.
type streamAdmissionGuard struct {
	lease streamAdmissionLease
	held  bool
}

func (g *streamAdmissionGuard) Acquire(deps TrackPublishDeps, campaignID uuid.UUID) (filter.FilterRejectKind, bool) {
	lease, kind, ok := deps.Reserve(campaignID)
	if !ok {
		return kind, false
	}
	g.lease = lease
	g.held = true
	return 0, true
}

func (g *streamAdmissionGuard) Release() {
	if g.held {
		g.lease.Release()
		g.held = false
	}
}

func (g *streamAdmissionGuard) LeasePtr() *streamAdmissionLease {
	if !g.held {
		return nil
	}
	return &g.lease
}

// httpTrackRejectProducerOverload writes 503 after PublishAcceptedOrRollback failed (rollback already applied).
func httpTrackRejectProducerOverload(w http.ResponseWriter, evt *domain.Event) int {
	recordHTTPFilterReject(filterRejectProducerOverload, evt)
	spec := filterRejectSpecs[filterRejectProducerOverload]
	w.Header().Set("Retry-After", "1")
	http.Error(w, spec.body, spec.status)
	domain.EventPool.Put(evt)
	return spec.status
}
