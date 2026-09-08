package ingest

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
)

// streamAdmissionLease holds one TryReserve slot until Release (reject path) or Clear (successful enqueue).
type streamAdmissionLease struct {
	release func()
}

func (l *streamAdmissionLease) Release() {
	if l != nil && l.release != nil {
		l.release()
		l.release = nil
	}
}

func (l *streamAdmissionLease) Clear() {
	if l != nil {
		l.release = nil
	}
}

type streamAdmissionTarget interface {
	tryReserve(admissionPct int) bool
	releaseReserve()
	queueDepthForMetric() int
	recordQueueDepth()
	recordRejected()
}

type streamProducerAdmissionTarget struct {
	producer *StreamProducer
	metrics  *streamAdmissionMetrics
}

func (t streamProducerAdmissionTarget) tryReserve(admissionPct int) bool {
	return t.producer.TryReserve(admissionPct)
}

func (t streamProducerAdmissionTarget) releaseReserve() {
	t.producer.ReleaseReserve()
}

func (t streamProducerAdmissionTarget) queueDepthForMetric() int {
	return t.producer.QueueDepth()
}

func (t streamProducerAdmissionTarget) recordQueueDepth() {
	if t.metrics != nil {
		t.metrics.queueDepth.Set(float64(t.queueDepthForMetric()))
	}
}

func (t streamProducerAdmissionTarget) recordRejected() {
	if t.metrics != nil {
		t.metrics.rejected.Inc()
	}
}

type brokerAdmissionTarget struct {
	broker  *BrokerProducer
	metrics *streamAdmissionMetrics
}

func (t brokerAdmissionTarget) tryReserve(admissionPct int) bool {
	return t.broker.TryReserve(admissionPct)
}

func (t brokerAdmissionTarget) releaseReserve() {
	t.broker.ReleaseReserve()
}

func (t brokerAdmissionTarget) queueDepthForMetric() int {
	return t.broker.PendingCount()
}

func (t brokerAdmissionTarget) recordQueueDepth() {
	if t.metrics != nil {
		t.metrics.queueDepth.Set(float64(t.queueDepthForMetric()))
	}
}

func (t brokerAdmissionTarget) recordRejected() {
	if t.metrics != nil {
		t.metrics.rejected.Inc()
	}
}

func streamAdmissionTargetFor(
	sharder Sharder,
	producers []*StreamProducer,
	brokers *BrokerProducerSet,
	campaignID uuid.UUID,
) (streamAdmissionTarget, bool) {
	if brokers != nil {
		idx, bp := brokers.Pick(campaignID)
		if bp != nil {
			return brokerAdmissionTarget{
				broker:  bp,
				metrics: brokerAdmissionMetricsFor(idx, brokers.Len() > 1),
			}, true
		}
	}
	if sharder == nil || len(producers) == 0 {
		return nil, false
	}
	shard := sharder.GetShard(campaignID)
	if shard < 0 || shard >= len(producers) {
		return nil, false
	}
	p := producers[shard]
	if p == nil {
		return nil, false
	}
	return streamProducerAdmissionTarget{producer: p, metrics: streamAdmissionMetricsForShard(shard)}, true
}

// tryAcquireStreamAdmission reserves producer queue headroom before Lua debit.
// requirePublisher true when defer-stream (fcap:ignored): fail filterRejectInfra if no StreamProducer/BrokerProducer.
// STREAM_PRODUCER_ADMISSION_PCT (default 85): occupied >= limit -> filterRejectProducerOverload 503, no debit.
//
// Verify:
// go test ./internal/ingest/ -short -run TestStreamProducerAdmission -count=1
func tryAcquireStreamAdmission(
	cfg *config.Config,
	sharder Sharder,
	producers []*StreamProducer,
	brokers *BrokerProducerSet,
	campaignID uuid.UUID,
	requirePublisher bool,
) (streamAdmissionLease, filter.FilterRejectKind, bool) {
	if requirePublisher && !trackIngestPublisherReady(sharder, producers, brokers, campaignID) {
		return streamAdmissionLease{}, filterRejectInfra, false
	}
	if cfg == nil || cfg.StreamProducerAdmissionPct <= 0 {
		return streamAdmissionLease{}, 0, true
	}
	target, ok := streamAdmissionTargetFor(sharder, producers, brokers, campaignID)
	if !ok {
		return streamAdmissionLease{}, 0, true
	}
	target.recordQueueDepth()
	if !target.tryReserve(cfg.StreamProducerAdmissionPct) {
		target.recordRejected()
		telemetry.RecordRejected()
		return streamAdmissionLease{}, filterRejectProducerOverload, false
	}
	lease := streamAdmissionLease{
		release: target.releaseReserve,
	}
	return lease, 0, true
}

func tryAcquireStreamAdmissionForFilter(
	cfg *config.Config,
	sharder Sharder,
	streamProducers []*StreamProducer,
	brokerProducers *BrokerProducerSet,
	campaignID uuid.UUID,
	filterEngine *FilterEngine,
) (streamAdmissionLease, filter.FilterRejectKind, bool) {
	return tryAcquireStreamAdmission(cfg, sharder, streamProducers, brokerProducers, campaignID, trackIngestRequiresPublisher(filterEngine))
}

func (h *AdsPacketHandler) tryAcquireStreamAdmission(campaignID uuid.UUID) (streamAdmissionLease, filter.FilterRejectKind, bool) {
	if h == nil {
		return streamAdmissionLease{}, 0, true
	}
	return tryAcquireStreamAdmissionForFilter(h.cfg, h.sharder, h.streamProducers, h.brokerProducers, campaignID, h.filterEngine)
}

func rejectIfStreamProducerOverloaded(
	cfg *config.Config,
	sharder Sharder,
	producers []*StreamProducer,
	brokers *BrokerProducerSet,
	campaignID uuid.UUID,
) (filter.FilterRejectKind, bool) {
	if cfg == nil || cfg.StreamProducerAdmissionPct <= 0 {
		return 0, false
	}
	target, ok := streamAdmissionTargetFor(sharder, producers, brokers, campaignID)
	if !ok {
		return 0, false
	}
	target.recordQueueDepth()
	pressurePct := 0
	switch t := target.(type) {
	case streamProducerAdmissionTarget:
		pressurePct = t.producer.QueuePressurePct()
	case brokerAdmissionTarget:
		pressurePct = t.broker.QueuePressurePct()
	}
	if pressurePct < cfg.StreamProducerAdmissionPct {
		return 0, false
	}
	target.recordRejected()
	telemetry.RecordRejected()
	return filterRejectProducerOverload, true
}

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
	return filterEngine.StreamDeferredToProducer()
}

// publishAcceptedTrackIngress async-enqueues one accepted event. Broker lane preferred when wired;
// else shard-indexed StreamProducer. With TryReserve lease: EnqueueReserved/ProcessReserved; lease.Clear on success.
// Deferred mode (fcap:ignored) without publisher -> false before enqueue; increments post_debit_rejected if miswired after debit.
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
