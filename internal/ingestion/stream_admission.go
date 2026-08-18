package ingestion

import (
	"strconv"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
)

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
	shardLabel() string
}

type streamProducerAdmissionTarget struct {
	producer *StreamProducer
	shard    string
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

func (t streamProducerAdmissionTarget) shardLabel() string {
	return t.shard
}

type brokerAdmissionTarget struct {
	broker *BrokerProducer
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

func (t brokerAdmissionTarget) shardLabel() string {
	return "broker"
}

func streamAdmissionTargetFor(
	sharder Sharder,
	producers []*StreamProducer,
	broker *BrokerProducer,
	campaignID uuid.UUID,
) (streamAdmissionTarget, bool) {
	if broker != nil {
		return brokerAdmissionTarget{broker: broker}, true
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
	return streamProducerAdmissionTarget{producer: p, shard: strconv.Itoa(shard)}, true
}

func tryAcquireStreamAdmission(
	cfg *config.Config,
	sharder Sharder,
	producers []*StreamProducer,
	broker *BrokerProducer,
	campaignID uuid.UUID,
) (streamAdmissionLease, filterRejectKind, bool) {
	if cfg == nil || cfg.StreamProducerAdmissionPct <= 0 {
		return streamAdmissionLease{}, 0, true
	}
	target, ok := streamAdmissionTargetFor(sharder, producers, broker, campaignID)
	if !ok {
		return streamAdmissionLease{}, 0, true
	}
	metrics.StreamProducerQueueDepth.WithLabelValues(target.shardLabel()).Set(float64(target.queueDepthForMetric()))
	if !target.tryReserve(cfg.StreamProducerAdmissionPct) {
		metrics.StreamProducerAdmissionRejectedTotal.WithLabelValues(target.shardLabel()).Inc()
		telemetry.RecordRejected()
		return streamAdmissionLease{}, filterRejectProducerOverload, false
	}
	lease := streamAdmissionLease{
		release: target.releaseReserve,
	}
	return lease, 0, true
}

func (h *AdsPacketHandler) tryAcquireStreamAdmission(campaignID uuid.UUID) (streamAdmissionLease, filterRejectKind, bool) {
	if h == nil {
		return streamAdmissionLease{}, 0, true
	}
	return tryAcquireStreamAdmission(h.cfg, h.sharder, h.streamProducers, h.brokerProducer, campaignID)
}

func rejectIfStreamProducerOverloaded(
	cfg *config.Config,
	sharder Sharder,
	producers []*StreamProducer,
	broker *BrokerProducer,
	campaignID uuid.UUID,
) (filterRejectKind, bool) {
	if cfg == nil || cfg.StreamProducerAdmissionPct <= 0 {
		return 0, false
	}
	target, ok := streamAdmissionTargetFor(sharder, producers, broker, campaignID)
	if !ok {
		return 0, false
	}
	metrics.StreamProducerQueueDepth.WithLabelValues(target.shardLabel()).Set(float64(target.queueDepthForMetric()))
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
	metrics.StreamProducerAdmissionRejectedTotal.WithLabelValues(target.shardLabel()).Inc()
	telemetry.RecordRejected()
	return filterRejectProducerOverload, true
}
