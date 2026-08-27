package ingestion

import (
	"context"
	"net/http"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

func trackIngestPublisherReady(sharder Sharder, producers []*StreamProducer, brokers *BrokerProducerSet, campaignID uuid.UUID) bool {
	if brokers != nil {
		if _, bp := brokers.Pick(campaignID); bp != nil {
			return true
		}
	}
	if sharder == nil || len(producers) == 0 {
		return false
	}
	shard := sharder.GetShard(campaignID)
	if shard < 0 || shard >= len(producers) {
		return false
	}
	return producers[shard] != nil
}

func trackIngestRequiresPublisher(filterEngine *FilterEngine) bool {
	if filterEngine == nil {
		return false
	}
	for _, filter := range filterEngine.filters {
		uf, ok := filter.(*UnifiedFilter)
		if ok && uf.StreamDeferredToProducer() {
			return true
		}
	}
	return false
}

func publishAcceptedTrackIngress(
	sharder Sharder,
	streamProducers []*StreamProducer,
	brokerProducers *BrokerProducerSet,
	filterEngine *FilterEngine,
	evt *domain.Event,
	lease *streamAdmissionLease,
) bool {
	if evt == nil {
		return true
	}
	deferred := trackIngestRequiresPublisher(filterEngine)
	if deferred && !trackIngestPublisherReady(sharder, streamProducers, brokerProducers, evt.CampaignID) {
		metrics.StreamProducerPostDebitRejectedTotal.Inc()
		return false
	}
	hasLease := lease != nil && lease.release != nil
	if brokerProducers != nil {
		_, bp := brokerProducers.Pick(evt.CampaignID)
		if bp != nil {
			var err error
			if hasLease {
				err = bp.EnqueueReserved(evt)
			} else {
				err = bp.Enqueue(evt)
			}
			if err != nil {
				metrics.StreamProducerPostDebitRejectedTotal.Inc()
				return false
			}
			if lease != nil {
				lease.Clear()
			}
			return true
		}
	}
	if sharder == nil || len(streamProducers) == 0 {
		if deferred {
			metrics.StreamProducerPostDebitRejectedTotal.Inc()
			return false
		}
		return true
	}
	shard := sharder.GetShard(evt.CampaignID)
	if shard < 0 || shard >= len(streamProducers) {
		if deferred {
			metrics.StreamProducerPostDebitRejectedTotal.Inc()
			return false
		}
		return true
	}
	p := streamProducers[shard]
	if p == nil {
		if deferred {
			metrics.StreamProducerPostDebitRejectedTotal.Inc()
			return false
		}
		return true
	}
	var err error
	if hasLease {
		err = p.ProcessReserved(evt)
	} else {
		err = p.Process(evt)
	}
	if err != nil {
		metrics.StreamProducerPostDebitRejectedTotal.Inc()
		return false
	}
	if lease != nil {
		lease.Clear()
	}
	return true
}

func httpTrackRejectProducerOverload(ctx context.Context, w http.ResponseWriter, filterEngine *FilterEngine, evt *domain.Event, registry domain.CampaignRegistry) int {
	if filterEngine != nil {
		filterEngine.RollbackDebit(ctx, evt, registry)
	}
	recordHTTPFilterReject(filterRejectProducerOverload, evt)
	spec := filterRejectSpecs[filterRejectProducerOverload]
	w.Header().Set("Retry-After", "1")
	http.Error(w, spec.body, spec.status)
	domain.EventPool.Put(evt)
	return spec.status
}
