package stream

import (
	"ad-event-processor/internal/stream/auditlog"
	"ad-event-processor/internal/stream/breaker"
	"ad-event-processor/internal/stream/broker"
	"ad-event-processor/internal/stream/codec"
	"ad-event-processor/internal/stream/fraud"
	"ad-event-processor/internal/stream/recon"
)

type CircuitState = breaker.CircuitState

const (
	CircuitClosed   = breaker.CircuitClosed
	CircuitOpen     = breaker.CircuitOpen
	CircuitHalfOpen = breaker.CircuitHalfOpen
)

type (
	ReconciliationWorker  = recon.ReconciliationWorker
	Snapshot              = recon.Snapshot
	SnapshotReplicator    = recon.SnapshotReplicator
	ClickHouseConn        = recon.ClickHouseConn
	PostgresConn          = recon.PostgresConn
	BrokerStreamConsumer  = broker.BrokerStreamConsumer
	BrokerReconcileConfig = broker.BrokerReconcileConfig
	BrokerReconcileWorker = broker.BrokerReconcileWorker
)

var (
	NewReconciliationWorker  = recon.NewReconciliationWorker
	ApplyRuntimeAutotune     = recon.ApplyRuntimeAutotune
	DefaultMaxWorkers        = recon.DefaultMaxWorkers
	NewSnapshotReplicator    = recon.NewSnapshotReplicator
	NewBrokerStreamConsumer  = broker.NewBrokerStreamConsumer
	NewBrokerReconcileWorker = broker.NewBrokerReconcileWorker
	StartFraudLagPublisher   = fraud.StartFraudLagPublisher
)

func UnsafeBytes(s string) []byte {
	return codec.UnsafeBytes(s)
}

func AuditLogSampleMaskFromConfig(cfgVal int) uint64 {
	return auditlog.SampleMaskFromConfig(cfgVal)
}
