package ingest

import (
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/stream"
)

type (
	SpendSyncBatchResult      = stream.SpendSyncBatchResult
	SpendSyncTransport        = stream.SpendSyncTransport
	SpendSyncProducer         = stream.SpendSyncProducer
	ConversionPayoutApplier   = stream.ConversionPayoutApplier
	ProcessorWeightController = stream.ProcessorWeightController
	SettlementWorker          = stream.SettlementWorker
	BrokerStreamConsumer      = stream.BrokerStreamConsumer
	BrokerReconcileConfig     = stream.BrokerReconcileConfig
	BrokerReconcileWorker     = stream.BrokerReconcileWorker
	GeoIPUpdaterConfig        = filter.GeoIPUpdaterConfig
)

var (
	ApplyClickHouseIngestPolicy         = stream.ApplyClickHouseIngestPolicy
	ClickHouseSpoolConfigFromConfig     = stream.ClickHouseSpoolConfigFromConfig
	InitProcessorClickHouseIngestPolicy = stream.InitProcessorClickHouseIngestPolicy
	NewProcessorPostgresGate            = stream.NewProcessorPostgresGate
	NewProcessorClickHouseGate          = stream.NewProcessorClickHouseGate
	NewConversionPayoutApplier          = stream.NewConversionPayoutApplier
	NewSettlementStore                  = stream.NewSettlementStore
	NewSettlementWorker                 = stream.NewSettlementWorker
	WrapEventStoreAfterBatch            = stream.WrapEventStoreAfterBatch
	WrapEventStoreBeforeBatch           = stream.WrapEventStoreBeforeBatch
	NewProcessorWeightController        = stream.NewProcessorWeightController
	ProcessorWeightConfigFromApp        = stream.ProcessorWeightConfigFromApp
	NewSpendSyncProducer                = stream.NewSpendSyncProducer
	NewBrokerStreamConsumer             = stream.NewBrokerStreamConsumer
	NewBrokerReconcileWorker            = stream.NewBrokerReconcileWorker
	ProcessorStreamLagSec               = stream.ProcessorStreamLagSec
	StartFraudLagPublisher              = stream.StartFraudLagPublisher
	NewGeoIPUpdater                     = filter.NewGeoIPUpdater
	ErrCHSpoolMaxSegments               = stream.ErrCHSpoolMaxSegments
)

const (
	ProcessorPgReserve = stream.ProcessorPgReserve
)

var errCHSpoolMaxSegments = stream.ErrCHSpoolMaxSegments
