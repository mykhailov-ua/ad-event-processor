package ingest

import (
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/stream"
	"ad-event-processor/internal/track"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	filter.SetWriteAuditLog(writeAuditLog)
}

const (
	luaMetricsSampleMask      = filter.LuaMetricsSampleMask
	auditLogSampleMaskDefault = 127
	fraudSignalL2Weak         = filter.FraudSignalL2Weak
	defaultLatencyRingCap     = filter.DefaultLatencyRingCap
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

func cachedUnixMilliNow() int64 {
	return filter.CachedUnixMilliNow()
}

func getCampaignFromEvent(registry domain.CampaignRegistry, evt *domain.Event) (*domain.Campaign, bool) {
	return filter.GetCampaignFromEvent(registry, evt)
}

func cgnatBypassForCampaign(
	globalBypass bool,
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	carrierTable *MobileCarrierASNTable,
	lookup filter.ASNLookup,
	ip string,
	signal string,
) bool {
	return filter.CgnatBypassForCampaign(globalBypass, registry, campaignID, carrierTable, lookup, ip, signal)
}

func IngressDayKey(buf []byte, regionCode uint8, customerID uuid.UUID, dateStr string) []byte {
	return filter.IngressDayKey(buf, regionCode, customerID, dateStr)
}

func histogramSampleMaskFromConfig(cfgVal int) uint64 {
	return filter.HistogramSampleMaskFromConfig(cfgVal)
}

func cachedUnixMilliAnyLoad() any {
	return filter.CachedUnixMilliAnyLoad()
}

func ShouldSampleHistogram(seq uint64, mask uint64) bool {
	return filter.ShouldSampleHistogram(seq, mask)
}

func observeHistogramSampled(seq *atomic.Uint64, mask uint64, observer prometheus.Observer, startMono int64) {
	filter.ObserveHistogramSampled(seq, mask, observer, startMono)
}

func monoElapsedSeconds(start int64) float64 {
	return filter.MonoElapsedSeconds(start)
}

var MonoElapsedSeconds = filter.MonoElapsedSeconds

func shouldSampleLuaMetrics(seq uint64) bool {
	return filter.ShouldSampleLuaMetrics(seq)
}

func auditLogSampleMaskFromConfig(cfgVal int) uint64 {
	return stream.AuditLogSampleMaskFromConfig(cfgVal)
}

func applyMobileBiometricSummary(evt *domain.Event) {
	filter.ApplyMobileBiometricSummary(evt)
}

func parseTCPSigHeader(b []byte) (uint32, bool) {
	return filter.ParseTCPSigHeader(b)
}

func uaMatchesInAppWebView(ua string) bool {
	return filter.UAMatchesInAppWebView(ua)
}

func asnLookupFromGeo(geo GeoProvider) filter.ASNLookup {
	return filter.AsnLookupFromGeo(geo)
}

const (
	LocalQuantaOff    = stream.LocalQuantaOff
	LocalQuantaShadow = stream.LocalQuantaShadow
	LocalQuantaLive   = stream.LocalQuantaLive
	CircuitClosed     = stream.CircuitClosed
	CircuitOpen       = stream.CircuitOpen
	CircuitHalfOpen   = stream.CircuitHalfOpen
)

const (
	filterRejectEmergencyBreaker   = filter.FilterRejectEmergencyBreaker
	filterRejectRateLimit          = filter.FilterRejectRateLimit
	filterRejectDuplicate          = filter.FilterRejectDuplicate
	filterRejectBudget             = filter.FilterRejectBudget
	filterRejectPacing             = filter.FilterRejectPacing
	filterRejectFreq               = filter.FilterRejectFreq
	filterRejectGeo                = filter.FilterRejectGeo
	filterRejectSchedule           = filter.FilterRejectSchedule
	filterRejectCampaignNotFound   = filter.FilterRejectCampaignNotFound
	filterRejectBidFloor           = filter.FilterRejectBidFloor
	filterRejectTimeout            = filter.FilterRejectTimeout
	filterRejectFraud              = filter.FilterRejectFraud
	filterRejectConsent            = filter.FilterRejectConsent
	filterRejectInfra              = filter.FilterRejectInfra
	filterRejectLicenseExpired     = filter.FilterRejectLicenseExpired
	filterRejectDailyQuotaExceeded = filter.FilterRejectDailyQuotaExceeded
	filterRejectPlacementBlocked   = filter.FilterRejectPlacementBlocked
	filterRejectSegmentExcluded    = filter.FilterRejectSegmentExcluded
	filterRejectSegmentNotIncluded = filter.FilterRejectSegmentNotIncluded
	filterRejectRegistryStale      = filter.FilterRejectRegistryStale
	filterRejectShardUnavailable   = filter.FilterRejectShardUnavailable
	filterRejectProducerOverload   = filter.FilterRejectProducerOverload
	filterRejectFraudBlocked       = filter.FilterRejectFraudBlocked
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

var cidrFeedNames = filter.CIDRFeedNames

var CIDRFeedNames = filter.CIDRFeedNames

type cidrNode = filter.CIDRNode

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

func osFingerprintMismatch(ua string, ttl uint8, windowSet uint8, window uint16) bool {
	return filter.OsFingerprintMismatch(ua, ttl, windowSet, window)
}

func tcpSynSigMismatch(ua string, sig uint32) bool {
	return filter.TcpSynSigMismatch(ua, sig)
}

func hashTCPSynFields(ttl uint8, window uint16, mss uint8, doff uint8) uint32 {
	return filter.HashTCPSynFields(ttl, window, mss, doff)
}

var HashTCPSynFields = filter.HashTCPSynFields
