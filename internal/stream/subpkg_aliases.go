package stream

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/stream/auditlog"
	"ad-event-processor/internal/stream/breaker"
	"ad-event-processor/internal/stream/broker"
	"ad-event-processor/internal/stream/codec"
	"ad-event-processor/internal/stream/fraud"
	"ad-event-processor/internal/stream/recon"
	"ad-event-processor/pkg/logger"
)

type (
	ByteSliceValue          = codec.ByteSliceValue
	CircuitBreaker          = breaker.CircuitBreaker
	CircuitState            = breaker.CircuitState
	BrokerClient            = broker.BrokerClient
	BrokerProducer          = broker.BrokerProducer
	BrokerProducerConfig    = broker.BrokerProducerConfig
	BrokerProducerSet       = broker.BrokerProducerSet
	BrokerConsumerConfig    = broker.BrokerConsumerConfig
	BrokerStreamConsumer    = broker.BrokerStreamConsumer
	BrokerReconcileConfig   = broker.BrokerReconcileConfig
	BrokerReconcileWorker   = broker.BrokerReconcileWorker
	FraudBrokerSink         = broker.FraudBrokerSink
	FraudStreamWriter       = fraud.FraudStreamWriter
	FraudBackpressureConfig = fraud.FraudBackpressureConfig
	ReconciliationWorker    = recon.ReconciliationWorker
	Snapshot                = recon.Snapshot
	SnapshotReplicator      = recon.SnapshotReplicator
	ClickHouseConn          = recon.ClickHouseConn
	PostgresConn            = recon.PostgresConn
)

const (
	CircuitClosed    = breaker.CircuitClosed
	CircuitOpen      = breaker.CircuitOpen
	CircuitHalfOpen  = breaker.CircuitHalfOpen
	MicroUnitFactor  = codec.MicroUnitFactor
	FraudAggForceKey = fraud.FraudAggForceKey
)

var (
	ErrRingBufferFull            = broker.ErrRingBufferFull
	ErrProducerClosed            = broker.ErrProducerClosed
	ErrBrokerPayloadUnrecognized = broker.ErrBrokerPayloadUnrecognized
	ErrFraudBrokerSinkConfig     = broker.ErrFraudBrokerSinkConfig
)

var (
	NewCircuitBreaker                   = breaker.NewCircuitBreaker
	DefaultBrokerProducerConfig         = broker.DefaultBrokerProducerConfig
	NewBrokerProducer                   = broker.NewBrokerProducer
	NewBrokerProducerSet                = broker.NewBrokerProducerSet
	NewBrokerStreamConsumer             = broker.NewBrokerStreamConsumer
	NewBrokerReconcileWorker            = broker.NewBrokerReconcileWorker
	NewFraudBrokerSink                  = broker.NewFraudBrokerSink
	NewFraudBrokerSinkWithClient        = broker.NewFraudBrokerSinkWithClient
	NewFraudStreamWriter                = fraud.NewFraudStreamWriter
	NewFraudStreamWriterNearFullForTest = fraud.NewFraudStreamWriterNearFullForTest
	ReadFraudAggForce                   = fraud.ReadFraudAggForce
	NewReconciliationWorker             = recon.NewReconciliationWorker
	ApplyRuntimeAutotune                = recon.ApplyRuntimeAutotune
	DefaultMaxWorkers                   = recon.DefaultMaxWorkers
	NewSnapshotReplicator               = recon.NewSnapshotReplicator
	ParseBrokerPayloadStream            = broker.ParseBrokerPayloadStream
	ParseBrokerPayload                  = broker.ParseBrokerPayload
	ParseIPv6To128                      = fraud.ParseIPv6To128
	EnqueueFraudReject                  = fraud.EnqueueFraudReject
	StartFraudBackpressureWatcher       = fraud.StartFraudBackpressureWatcher
	PublishFraudConsumerLag             = fraud.PublishFraudConsumerLag
	StartFraudLagPublisher              = fraud.StartFraudLagPublisher
	SliceToMap                          = codec.SliceToMap
	UnsafeString                        = codec.UnsafeString
	UnsafeBytes                         = codec.UnsafeBytes
	DeepResetAdStreamEvent              = codec.DeepResetAdStreamEvent
	ClearAdStreamEvent                  = codec.ClearAdStreamEvent
	DeepResetAdDLQEvent                 = codec.DeepResetAdDLQEvent
)

func AuditLogSampleMaskFromConfig(cfgVal int) uint64 {
	return auditlog.SampleMaskFromConfig(cfgVal)
}

func WriteAuditLog(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
) {
	auditlog.Write(l, seq, sampleMask, shardID, evt)
}

func unsafeString(b []byte) string {
	return codec.UnsafeString(b)
}
