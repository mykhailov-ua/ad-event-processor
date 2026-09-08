package compat

import (
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/stream"
	"ad-event-processor/internal/track"

	"github.com/google/uuid"
)

func FraudReasonCode(id FraudReasonID) string {
	return filter.FraudReasonCode(id)
}

func CachedTimeIn(loc *time.Location) time.Time {
	return filter.CachedTimeIn(loc)
}

func CachedTimeUTC() time.Time {
	return filter.CachedTimeUTC()
}

func CachedHourUTC() int {
	return filter.CachedHourUTC()
}

func IngressDayKey(buf []byte, regionCode uint8, customerID uuid.UUID, dateStr string) []byte {
	return filter.IngressDayKey(buf, regionCode, customerID, dateStr)
}

func ShouldSampleHistogram(seq uint64, mask uint64) bool {
	return filter.ShouldSampleHistogram(seq, mask)
}

var MonoElapsedSeconds = filter.MonoElapsedSeconds

const (
	LocalQuantaOff    = stream.LocalQuantaOff
	LocalQuantaShadow = stream.LocalQuantaShadow
	LocalQuantaLive   = stream.LocalQuantaLive
	CircuitClosed     = stream.CircuitClosed
	CircuitOpen       = stream.CircuitOpen
	CircuitHalfOpen   = stream.CircuitHalfOpen
)

type (
	Registry                         = filter.Registry
	FlowRouter                       = filter.FlowRouter
	BudgetCacheWarmer                = filter.BudgetCacheWarmer
	DCASNTable                       = filter.DCASNTable
	GeoProvider                      = filter.GeoProvider
	SettingsWatcher                  = filter.SettingsWatcher
	MobileCarrierASNTable            = filter.MobileCarrierASNTable
	LicenseStateReader               = filter.LicenseStateReader
	LatencyRing                      = filter.LatencyRing
	CIDRTable                        = filter.CIDRTable
	CIDRNode                         = filter.CIDRNode
	ModeratorIPTable                 = filter.ModeratorIPTable
	CampaignFlowTable                = filter.CampaignFlowTable
	LocalQuotaCache                  = filter.LocalQuotaCache
	Sharder                          = domain.Sharder
	SubIDSlots                       = track.SubIDSlots
	FraudStreamWriter                = stream.FraudStreamWriter
	BrokerProducerSet                = stream.BrokerProducerSet
	StreamProducer                   = stream.StreamProducer
	StreamConsumer                   = stream.StreamConsumer
	ClickHouseStore                  = stream.ClickHouseStore
	ClickHouseSpool                  = stream.ClickHouseSpool
	ClickHouseSpoolConfig            = stream.ClickHouseSpoolConfig
	PostgresStore                    = stream.PostgresStore
	LocalQuantaLedger                = stream.LocalQuantaLedger
	LocalQuantaStrict                = stream.LocalQuantaStrict
	QuotaRefillWorker                = stream.QuotaRefillWorker
	QuotaRefillConfig                = stream.QuotaRefillConfig
	BudgetDeltaPublisher             = stream.BudgetDeltaPublisher
	BudgetDeltaPublisherConfig       = stream.BudgetDeltaPublisherConfig
	LocalQuantaStreamPublisher       = stream.LocalQuantaStreamPublisher
	LocalQuantaStreamPublisherConfig = stream.LocalQuantaStreamPublisherConfig
	LocalQuantaFlusher               = stream.LocalQuantaFlusher
	LocalClickIdemCache              = stream.LocalClickIdemCache
	BrokerProducer                   = stream.BrokerProducer
	BrokerProducerConfig             = stream.BrokerProducerConfig
	BrokerConsumerConfig             = stream.BrokerConsumerConfig
	FraudBrokerSink                  = stream.FraudBrokerSink
	StreamProducerConfig             = stream.StreamProducerConfig
	FraudBackpressureConfig          = stream.FraudBackpressureConfig
	CircuitState                     = stream.CircuitState
	ResidentialProxyRing             = filter.ResidentialProxyRing
	ResidentialProxyRow              = filter.ResidentialProxyRow
	ResidentialIntelTable            = filter.ResidentialIntelTable
	ProxyVPNTable                    = filter.ProxyVPNTable
	ProxyVPNBuilder                  = filter.ProxyVPNBuilder
	ProxyVPNSnapshot                 = filter.ProxyVPNSnapshot
	StaticSlotSharder                = domain.StaticSlotSharder
	FlowSelection                    = filter.FlowSelection
	FraudScoreBoostSnapshot          = filter.FraudScoreBoostSnapshot
	MaxMindProvider                  = filter.MaxMindProvider
	MockGeoProvider                  = filter.MockGeoProvider
	RegistryLicenseConfig            = filter.RegistryLicenseConfig
	HybridBalancer                   = filter.HybridBalancer
	FlowPathSnapshot                 = filter.FlowPathSnapshot
	FlowPath                         = filter.FlowPath
	FlowLanderEntry                  = filter.FlowLanderEntry
	FlowOfferEntry                   = filter.FlowOfferEntry
	FlowSelectContext                = filter.FlowSelectContext
	CampaignFlowRegistrySnapshot     = filter.CampaignFlowRegistrySnapshot
	CampaignMeta                     = filter.CampaignMeta
	DynamicConfig                    = filter.DynamicConfig
)

const CIDRFeedCount = filter.CIDRFeedCount

const (
	CIDRFeedAWS      = filter.CIDRFeedAWS
	CIDRFeedGCP      = filter.CIDRFeedGCP
	CIDRFeedAzure    = filter.CIDRFeedAzure
	CIDRFeedTor      = filter.CIDRFeedTor
	CIDRFeedOther    = filter.CIDRFeedOther
	FlushReasonPause = stream.FlushReasonPause
)

var CIDRFeedNames = filter.CIDRFeedNames

var (
	NewRegistry                          = filter.NewRegistry
	NewFlowRouter                        = filter.NewFlowRouter
	BanditSelect                         = filter.BanditSelect
	SelectSnapshot                       = filter.SelectSnapshot
	NewSettingsWatcher                   = filter.NewSettingsWatcher
	NewDCASNTable                        = filter.NewDCASNTable
	NewMobileCarrierASNTable             = filter.NewMobileCarrierASNTable
	NewLatencyRing                       = filter.NewLatencyRing
	NewCIDRTable                         = filter.NewCIDRTable
	BuildCIDRTableFromPrefixes           = filter.BuildCIDRTableFromPrefixes
	NewStreamProducer                    = stream.NewStreamProducer
	NewStreamProducerQueueForTest        = stream.NewStreamProducerQueueForTest
	NewStreamConsumer                    = stream.NewStreamConsumer
	NewFraudStreamWriter                 = stream.NewFraudStreamWriter
	NewFraudStreamWriterNearFullForTest  = stream.NewFraudStreamWriterNearFullForTest
	ReadFraudAggForce                    = stream.ReadFraudAggForce
	FraudAggForceKey                     = stream.FraudAggForceKey
	NewLocalQuantaLedger                 = stream.NewLocalQuantaLedger
	NewLocalQuantaStrict                 = stream.NewLocalQuantaStrict
	NewQuotaRefillWorker                 = stream.NewQuotaRefillWorker
	NewBudgetDeltaPublisher              = stream.NewBudgetDeltaPublisher
	NewLocalQuantaStreamPublisher        = stream.NewLocalQuantaStreamPublisher
	NewLocalQuantaStreamPublisherForTest = stream.NewLocalQuantaStreamPublisherForTest
	NewLocalQuotaCache                   = filter.NewLocalQuotaCache
	NewResidentialProxyRing              = filter.NewResidentialProxyRing
	DefaultResidentialProxyPolicyForTest = filter.DefaultResidentialProxyPolicyForTest
	ResidentialProxySignalForTest        = filter.ResidentialProxySignalForTest
	NewResidentialIntelTable             = filter.NewResidentialIntelTable
	NewProxyVPNTable                     = filter.NewProxyVPNTable
	NewStaticSlotSharder                 = domain.NewStaticSlotSharder
	NewLocalClickIdemCache               = stream.NewLocalClickIdemCache
	NewBrokerProducerSet                 = stream.NewBrokerProducerSet
	NewJumpHashSharder                   = domain.NewJumpHashSharder
	MigrationFenceKeyPrefix              = filter.MigrationFenceKeyPrefix
	SetStoreRetryPolicy                  = stream.SetStoreRetryPolicy
	NewBudgetCacheWarmer                 = filter.NewBudgetCacheWarmer
	NewClickHouseStore                   = stream.NewClickHouseStore
	NewPostgresStore                     = stream.NewPostgresStore
	NewPostgresStoreWithGate             = stream.NewPostgresStoreWithGate
	OpenClickHouseSpool                  = stream.OpenClickHouseSpool
	OpenClickHouseSpoolWithConfig        = stream.OpenClickHouseSpoolWithConfig
	DefaultClickHouseSpoolConfig         = stream.DefaultClickHouseSpoolConfig
	MarshalCHSpoolPayload                = stream.MarshalCHSpoolPayload
	NewMaxMindProvider                   = filter.NewMaxMindProvider
	NewGeoIPWatcher                      = filter.NewGeoIPWatcher
	NewDCASNFeedLoader                   = filter.NewDCASNFeedLoader
	ParseMobileCarrierASNs               = filter.ParseMobileCarrierASNs
	ResidentialProxyPolicyFromEnv        = filter.ResidentialProxyPolicyFromEnv
	NewConsentFilter                     = filter.NewConsentFilter
	FilterRedisOptions                   = filter.FilterRedisOptions
	FilterRedisReadTimeoutMs             = filter.FilterRedisReadTimeoutMs
	NewLocalQuantaFlusher                = stream.NewLocalQuantaFlusher
	FetchRecoveryDeltas                  = stream.FetchRecoveryDeltas
	SetRegistryQuantaFlushHook           = filter.SetRegistryQuantaFlushHook
	InvokeRegistryQuantaFlush            = filter.InvokeRegistryQuantaFlush
	LocalQuotaReturnScript               = stream.LocalQuotaReturnScript
	LocalQuantaStreamUsable              = stream.LocalQuantaStreamUsable
	SettingsPGSync                       = filter.SettingsPGSync
	NewLicenseFilter                     = filter.NewLicenseFilter
	NewVPPFilter                         = filter.NewVPPFilter
	NewHybridBalancer                    = filter.NewHybridBalancer
	DefaultBrokerProducerConfig          = stream.DefaultBrokerProducerConfig
	NewBrokerProducer                    = stream.NewBrokerProducer
	AdaptiveChunkSize                    = stream.AdaptiveChunkSize
	AdaptiveChunkSizeStrict              = stream.AdaptiveChunkSizeStrict
	NewFraudBrokerSink                   = stream.NewFraudBrokerSink
	StartFraudBackpressureWatcher        = stream.StartFraudBackpressureWatcher
	NewCIDRFeedLoader                    = filter.NewCIDRFeedLoader
	NewProxyVPNFeedLoader                = filter.NewProxyVPNFeedLoader
	NewModeratorIPTable                  = filter.NewModeratorIPTable
	NewModeratorIntelFeedLoader          = filter.NewModeratorIntelFeedLoader
	NewCampaignFlowTable                 = filter.NewCampaignFlowTable
	NewCampaignFlowRegistrySnapshot      = filter.NewCampaignFlowRegistrySnapshot
	NewCampaignFlowSync                  = filter.NewCampaignFlowSync
	ModeratorIntelFeedFileName           = filter.ModeratorIntelFeedFileName
	ModeratorIntelSigFileName            = filter.ModeratorIntelSigFileName
)

var HashTCPSynFields = filter.HashTCPSynFields
