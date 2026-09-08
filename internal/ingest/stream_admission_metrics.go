package ingest

import (
	"strconv"
	"sync/atomic"

	"ad-event-processor/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// Upper bound for stream/broker producer shard labels (production Redis topology is <=16).
const admissionMetricsMaxShards = 64

type streamAdmissionMetrics struct {
	queueDepth prometheus.Gauge
	rejected   prometheus.Counter
}

var (
	streamAdmissionMetricsSlots [admissionMetricsMaxShards]atomic.Pointer[streamAdmissionMetrics]
	brokerAdmissionMetricsSlots [admissionMetricsMaxShards]atomic.Pointer[streamAdmissionMetrics]
)

func streamAdmissionMetricsForShard(shard int) *streamAdmissionMetrics {
	slot := shard
	if slot < 0 || slot >= admissionMetricsMaxShards {
		slot = 0
	}
	if m := streamAdmissionMetricsSlots[slot].Load(); m != nil {
		return m
	}
	label := shardLabelString(shard)
	m := &streamAdmissionMetrics{
		queueDepth: metrics.StreamProducerQueueDepth.WithLabelValues(label),
		rejected:   metrics.StreamProducerAdmissionRejectedTotal.WithLabelValues(label),
	}
	for {
		if existing := streamAdmissionMetricsSlots[slot].Load(); existing != nil {
			return existing
		}
		if streamAdmissionMetricsSlots[slot].CompareAndSwap(nil, m) {
			return m
		}
	}
}

func brokerAdmissionMetricsFor(idx int, multi bool) *streamAdmissionMetrics {
	slot := 0
	label := "broker"
	if multi {
		slot = idx + 1
		label = "broker-" + shardLabelString(idx)
	}
	if slot < 0 || slot >= admissionMetricsMaxShards {
		slot = 0
	}
	if m := brokerAdmissionMetricsSlots[slot].Load(); m != nil {
		return m
	}
	m := &streamAdmissionMetrics{
		queueDepth: metrics.StreamProducerQueueDepth.WithLabelValues(label),
		rejected:   metrics.StreamProducerAdmissionRejectedTotal.WithLabelValues(label),
	}
	for {
		if existing := brokerAdmissionMetricsSlots[slot].Load(); existing != nil {
			return existing
		}
		if brokerAdmissionMetricsSlots[slot].CompareAndSwap(nil, m) {
			return m
		}
	}
}

func shardLabelString(shard int) string {
	return strconv.Itoa(shard)
}
