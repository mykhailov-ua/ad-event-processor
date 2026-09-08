package ingest

import (
	"sync"

	"ad-event-processor/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

type streamAdmissionMetrics struct {
	queueDepth prometheus.Gauge
	rejected   prometheus.Counter
}

var (
	streamAdmissionMetricsCache sync.Map
	brokerAdmissionMetricsCache sync.Map
)

func streamAdmissionMetricsForShard(shard int) *streamAdmissionMetrics {
	if v, ok := streamAdmissionMetricsCache.Load(shard); ok {
		return v.(*streamAdmissionMetrics)
	}
	label := shardLabelString(shard)
	m := &streamAdmissionMetrics{
		queueDepth: metrics.StreamProducerQueueDepth.WithLabelValues(label),
		rejected:   metrics.StreamProducerAdmissionRejectedTotal.WithLabelValues(label),
	}
	actual, _ := streamAdmissionMetricsCache.LoadOrStore(shard, m)
	return actual.(*streamAdmissionMetrics)
}

func brokerAdmissionMetricsFor(idx int, multi bool) *streamAdmissionMetrics {
	key := idx
	if !multi {
		key = -1
	}
	if v, ok := brokerAdmissionMetricsCache.Load(key); ok {
		return v.(*streamAdmissionMetrics)
	}
	label := "broker"
	if multi {
		label = "broker-" + shardLabelString(idx)
	}
	m := &streamAdmissionMetrics{
		queueDepth: metrics.StreamProducerQueueDepth.WithLabelValues(label),
		rejected:   metrics.StreamProducerAdmissionRejectedTotal.WithLabelValues(label),
	}
	actual, _ := brokerAdmissionMetricsCache.LoadOrStore(key, m)
	return actual.(*streamAdmissionMetrics)
}

func shardLabelString(shard int) string {
	var scratch [8]byte
	return unsafeString(appendInt64(scratch[:0], int64(shard)))
}
