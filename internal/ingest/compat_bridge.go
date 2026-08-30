package ingest

import (
	"net/netip"
	"time"

	"ad-event-processor/internal/domain"
	cp "ad-event-processor/internal/ingest/compat"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	FraudReasonID = cp.FraudReasonID
)

type (
	CampaignRepo                 = cp.CampaignRepo
	CustomerRepo                 = cp.CustomerRepo
	QuotaRepo                    = cp.QuotaRepo
	SlotMapRepo                  = cp.SlotMapRepo
	SlotMigrationRepo            = cp.SlotMigrationRepo
	CampaignRoutingRepo          = cp.CampaignRoutingRepo
	SyncWorker                   = cp.SyncWorker
	SpendFlushItem               = cp.SpendFlushItem
	SpendFlushOutcome            = cp.SpendFlushOutcome
	PendingRollup                = cp.PendingRollup
	BudgetReconSnapshot          = cp.BudgetReconSnapshot
	BudgetInvariantSnapshot      = cp.BudgetInvariantSnapshot
	SlotOverride                 = cp.SlotOverride
	ReserveChunkResult           = cp.ReserveChunkResult
	MarginEconomicsSplit         = cp.MarginEconomicsSplit
	ConsentStore                 = cp.ConsentStore
	CampaignKeyMigrator          = cp.CampaignKeyMigrator
	CampaignRedisKeyCatalog      = cp.CampaignRedisKeyCatalog
	SlotMigrationDualWriteConfig = cp.SlotMigrationDualWriteConfig
	SlotMigrationDelta           = cp.SlotMigrationDelta
	OpsSlotMapResponse           = cp.OpsSlotMapResponse
	CTVSettlementResult          = cp.CTVSettlementResult
	BudgetDeltaAggregator        = cp.BudgetDeltaAggregator
	RtbBidShadeInput             = cp.RtbBidShadeInput
	RtbBidShadeOutput            = cp.RtbBidShadeOutput
)

type (
	UDPControlLimits        = cp.UDPControlLimits
	IngressQuotaMap         = cp.IngressQuotaMap
	UDPHeader               = cp.UDPHeader
	UDPNodeWeight           = cp.UDPNodeWeight
	UDPConfigRequestPayload = cp.UDPConfigRequestPayload
	UDPNodeWeightJSON       = cp.UDPNodeWeightJSON
	TCPControlHeader        = cp.TCPControlHeader
	TCPAckPayload           = cp.TCPAckPayload
)

type (
	Registry                         = cp.Registry
	FlowRouter                       = cp.FlowRouter
	BudgetCacheWarmer                = cp.BudgetCacheWarmer
	DCASNTable                       = cp.DCASNTable
	GeoProvider                      = cp.GeoProvider
	SettingsWatcher                  = cp.SettingsWatcher
	MobileCarrierASNTable            = cp.MobileCarrierASNTable
	LicenseStateReader               = cp.LicenseStateReader
	LatencyRing                      = cp.LatencyRing
	CIDRTable                        = cp.CIDRTable
	CIDRNode                         = cp.CIDRNode
	ModeratorIPTable                 = cp.ModeratorIPTable
	CampaignFlowTable                = cp.CampaignFlowTable
	LocalQuotaCache                  = cp.LocalQuotaCache
	Sharder                          = cp.Sharder
	SubIDSlots                       = cp.SubIDSlots
	FraudStreamWriter                = cp.FraudStreamWriter
	BrokerProducerSet                = cp.BrokerProducerSet
	StreamProducer                   = cp.StreamProducer
	StreamConsumer                   = cp.StreamConsumer
	ClickHouseStore                  = cp.ClickHouseStore
	ClickHouseSpool                  = cp.ClickHouseSpool
	ClickHouseSpoolConfig            = cp.ClickHouseSpoolConfig
	PostgresStore                    = cp.PostgresStore
	LocalQuantaLedger                = cp.LocalQuantaLedger
	LocalQuantaStrict                = cp.LocalQuantaStrict
	QuotaRefillWorker                = cp.QuotaRefillWorker
	QuotaRefillConfig                = cp.QuotaRefillConfig
	BudgetDeltaPublisher             = cp.BudgetDeltaPublisher
	BudgetDeltaPublisherConfig       = cp.BudgetDeltaPublisherConfig
	LocalQuantaStreamPublisher       = cp.LocalQuantaStreamPublisher
	LocalQuantaStreamPublisherConfig = cp.LocalQuantaStreamPublisherConfig
	LocalQuantaFlusher               = cp.LocalQuantaFlusher
	LocalClickIdemCache              = cp.LocalClickIdemCache
	BrokerProducer                   = cp.BrokerProducer
	BrokerProducerConfig             = cp.BrokerProducerConfig
	BrokerConsumerConfig             = cp.BrokerConsumerConfig
	FraudBrokerSink                  = cp.FraudBrokerSink
	StreamProducerConfig             = cp.StreamProducerConfig
	FraudBackpressureConfig          = cp.FraudBackpressureConfig
	CircuitState                     = cp.CircuitState
	ResidentialProxyRing             = cp.ResidentialProxyRing
	ResidentialProxyRow              = cp.ResidentialProxyRow
	ResidentialIntelTable            = cp.ResidentialIntelTable
	ProxyVPNTable                    = cp.ProxyVPNTable
	ProxyVPNBuilder                  = cp.ProxyVPNBuilder
	ProxyVPNSnapshot                 = cp.ProxyVPNSnapshot
	StaticSlotSharder                = cp.StaticSlotSharder
	FlowSelection                    = cp.FlowSelection
	FraudScoreBoostSnapshot          = cp.FraudScoreBoostSnapshot
	MaxMindProvider                  = cp.MaxMindProvider
	MockGeoProvider                  = cp.MockGeoProvider
	RegistryLicenseConfig            = cp.RegistryLicenseConfig
	HybridBalancer                   = cp.HybridBalancer
	FlowPathSnapshot                 = cp.FlowPathSnapshot
	FlowPath                         = cp.FlowPath
	FlowLanderEntry                  = cp.FlowLanderEntry
	FlowOfferEntry                   = cp.FlowOfferEntry
	FlowSelectContext                = cp.FlowSelectContext
	CampaignFlowRegistrySnapshot     = cp.CampaignFlowRegistrySnapshot
	CampaignMeta                     = cp.CampaignMeta
	DynamicConfig                    = cp.DynamicConfig
)

const (
	FraudReasonNone                     = cp.FraudReasonNone
	FraudReasonDatacenterIP             = cp.FraudReasonDatacenterIP
	FraudReasonLowTTC                   = cp.FraudReasonLowTTC
	FraudReasonMissingImpTS             = cp.FraudReasonMissingImpTS
	FraudReasonL3Blocklist              = cp.FraudReasonL3Blocklist
	FraudReasonTLSBlocklist             = cp.FraudReasonTLSBlocklist
	FraudReasonDeviceMismatch           = cp.FraudReasonDeviceMismatch
	FraudReasonTCPMSSAnomaly            = cp.FraudReasonTCPMSSAnomaly
	FraudReasonTCPTunnelMSS             = cp.FraudReasonTCPTunnelMSS
	FraudReasonTCPSynOSMismatch         = cp.FraudReasonTCPSynOSMismatch
	FraudReasonJSONSerializationBot     = cp.FraudReasonJSONSerializationBot
	FraudReasonOSFingerprint            = cp.FraudReasonOSFingerprint
	FraudReasonIPv4Rotation             = cp.FraudReasonIPv4Rotation
	FraudReasonResidentialProxy         = cp.FraudReasonResidentialProxy
	FraudReasonAttestationMissing       = cp.FraudReasonAttestationMissing
	FraudReasonModeratorIP              = cp.FraudReasonModeratorIP
	FraudReasonSecFetchAnomaly          = cp.FraudReasonSecFetchAnomaly
	FraudReasonClientHintsMismatch      = cp.FraudReasonClientHintsMismatch
	FraudReasonTLSALPNMismatch          = cp.FraudReasonTLSALPNMismatch
	FraudReasonH2SettingsMismatch       = cp.FraudReasonH2SettingsMismatch
	FraudReasonH2PseudoOrder            = cp.FraudReasonH2PseudoOrder
	FraudReasonH2DowngradeArtifact      = cp.FraudReasonH2DowngradeArtifact
	FraudReasonHeaderOrderMismatch      = cp.FraudReasonHeaderOrderMismatch
	FraudReasonAcceptEncodingMismatch   = cp.FraudReasonAcceptEncodingMismatch
	FraudReasonAcceptLangGeoMismatch    = cp.FraudReasonAcceptLangGeoMismatch
	FraudReasonTLSJA4Mismatch           = cp.FraudReasonTLSJA4Mismatch
	FraudReasonBehaviorTelemetryMissing = cp.FraudReasonBehaviorTelemetryMissing
	FraudReasonBehaviorBezierBot        = cp.FraudReasonBehaviorBezierBot
)

const (
	ProxyVPNConnISP     = cp.ProxyVPNConnISP
	ProxyVPNConnHosting = cp.ProxyVPNConnHosting
	ProxyVPNConnVPN     = cp.ProxyVPNConnVPN
	ProxyVPNConnMobile  = cp.ProxyVPNConnMobile
)

const (
	UAFamilyWindows = cp.UAFamilyWindows
	UAFamilyMac     = cp.UAFamilyMac
	UAFamilyLinux   = cp.UAFamilyLinux
	UAFamilyMobile  = cp.UAFamilyMobile
	UAFamilyUnknown = cp.UAFamilyUnknown
)

const (
	FraudReasonCodeMissingImpTS  = cp.FraudReasonCodeMissingImpTS
	FraudReasonCodeLowTTC        = cp.FraudReasonCodeLowTTC
	FraudReasonCodeOSFingerprint = cp.FraudReasonCodeOSFingerprint
)

const (
	SlotCount                        = cp.SlotCount
	SlotMask                         = cp.SlotMask
	ConsentPurposeAdStorage          = cp.ConsentPurposeAdStorage
	ConsentPurposeAnalytics          = cp.ConsentPurposeAnalytics
	ConsentRedisKeyPrefix            = cp.ConsentRedisKeyPrefix
	ConsentDefaultUpdateChannel      = cp.ConsentDefaultUpdateChannel
	DefaultCampaignUpdateBrokerTopic = cp.DefaultCampaignUpdateBrokerTopic
	RegistryFullSyncPayload          = cp.RegistryFullSyncPayload
	DefaultSlotMapReloadTopic        = cp.DefaultSlotMapReloadTopic
	SlotMigrationDualWriteFlagKey    = cp.SlotMigrationDualWriteFlagKey
	DefaultRtbCatalogReloadChannel   = cp.DefaultRtbCatalogReloadChannel
)

const (
	UDPHeaderSize          = cp.UDPHeaderSize
	UDPMaxControlShards    = cp.UDPMaxControlShards
	UDPMsgQuotaEpoch       = cp.UDPMsgQuotaEpoch
	UDPMsgConfigSnapshot   = cp.UDPMsgConfigSnapshot
	UDPMsgConfigRequest    = cp.UDPMsgConfigRequest
	UDPMsgMigrationBarrier = cp.UDPMsgMigrationBarrier
	UDPFlagSnapshot        = cp.UDPFlagSnapshot
	UDPFlagNodeWeights     = cp.UDPFlagNodeWeights
	TCPControlHeaderSize   = cp.TCPControlHeaderSize
	TCPMsgSnapshot         = cp.TCPMsgSnapshot
	TCPMsgSnapshotRequest  = cp.TCPMsgSnapshotRequest
	TCPMsgAck              = cp.TCPMsgAck
)

const (
	LocalQuantaOff    = cp.LocalQuantaOff
	LocalQuantaShadow = cp.LocalQuantaShadow
	LocalQuantaLive   = cp.LocalQuantaLive
	CircuitClosed     = cp.CircuitClosed
	CircuitOpen       = cp.CircuitOpen
	CircuitHalfOpen   = cp.CircuitHalfOpen
)

const (
	CIDRFeedAWS      = cp.CIDRFeedAWS
	CIDRFeedGCP      = cp.CIDRFeedGCP
	CIDRFeedAzure    = cp.CIDRFeedAzure
	CIDRFeedTor      = cp.CIDRFeedTor
	CIDRFeedOther    = cp.CIDRFeedOther
	FlushReasonPause = cp.FlushReasonPause
)

const CampaignEpochKey = cp.CampaignEpochKey

const RtbFloorRedisKeyPrefix = cp.RtbFloorRedisKeyPrefix

const CIDRFeedCount = cp.CIDRFeedCount

var (
	ErrSegmentNotIncluded = cp.ErrSegmentNotIncluded
	ErrSegmentExcluded    = cp.ErrSegmentExcluded
)

var (
	ErrConsentDenied                = cp.ErrConsentDenied
	FraudReasonCodeDatacenterIP     = cp.FraudReasonCodeDatacenterIP
	FraudReasonCodeL3Blocklist      = cp.FraudReasonCodeL3Blocklist
	FraudReasonCodeIPv4Rotation     = cp.FraudReasonCodeIPv4Rotation
	FraudReasonCodeTCPSynOSMismatch = cp.FraudReasonCodeTCPSynOSMismatch
	ClassifyFilterErr               = cp.ClassifyFilterErr
	ParseASNLine                    = cp.ParseASNLine
)

var (
	CRC32Castagnoli       = cp.CRC32Castagnoli
	AppendCampaignHashTag = cp.AppendCampaignHashTag
	PlacementBlacklistKey = cp.PlacementBlacklistKey
)

var (
	NewCampaignRepo                     = cp.NewCampaignRepo
	NewCampaignRepoWithDB               = cp.NewCampaignRepoWithDB
	NewCustomerRepo                     = cp.NewCustomerRepo
	NewCustomerRepoWithDB               = cp.NewCustomerRepoWithDB
	NewQuotaRepo                        = cp.NewQuotaRepo
	NewSlotMapRepo                      = cp.NewSlotMapRepo
	NewSlotMigrationRepo                = cp.NewSlotMigrationRepo
	NewCampaignRoutingRepo              = cp.NewCampaignRoutingRepo
	NewSyncWorker                       = cp.NewSyncWorker
	MaxLedgerBatchSize                  = cp.MaxLedgerBatchSize
	BudgetLockTTLSeconds                = cp.BudgetLockTTLSeconds
	FetchBudgetReconSnapshot            = cp.FetchBudgetReconSnapshot
	ReadBudgetInvariant                 = cp.ReadBudgetInvariant
	ReadBudgetInvariants                = cp.ReadBudgetInvariants
	VerifyBudgetInvariant               = cp.VerifyBudgetInvariant
	AssertBudgetInvariant               = cp.AssertBudgetInvariant
	CampaignShardID                     = cp.CampaignShardID
	CampaignSlotIndex                   = cp.CampaignSlotIndex
	FilterCampaignIDsBySlot             = cp.FilterCampaignIDsBySlot
	ComputeMarginEconomicsSplit         = cp.ComputeMarginEconomicsSplit
	WriteMarginEconomicsLegs            = cp.WriteMarginEconomicsLegs
	TableFromRows                       = cp.TableFromRows
	CampaignFromDBRow                   = cp.CampaignFromDBRow
	CampaignFromGetCampaignFullRow      = cp.CampaignFromGetCampaignFullRow
	CampaignFromListActiveCampaignsRow  = cp.CampaignFromListActiveCampaignsRow
	EncodeQuotaEpochDatagramWithWeights = cp.EncodeQuotaEpochDatagramWithWeights
	MarshalEpochPayload                 = cp.MarshalEpochPayload
	NodeWeightsToJSON                   = cp.NodeWeightsToJSON
	ControlFailOpenEnabled              = cp.ControlFailOpenEnabled
	EdgeControlEqualizeWeights          = cp.EdgeControlEqualizeWeights
	EdgeControlDrainFrozen              = cp.EdgeControlDrainFrozen
	QuotaShardForCampaign               = cp.QuotaShardForCampaign
	EqualizeNodeWeights                 = cp.EqualizeNodeWeights
	LoadActiveSlotMap                   = cp.LoadActiveSlotMap
	ReloadStaticSlotMapIfChanged        = cp.ReloadStaticSlotMapIfChanged
	SlotMapShardTable                   = cp.SlotMapShardTable
	BumpMigrationFences                 = cp.BumpMigrationFences
	SetBudgetFrozen                     = cp.SetBudgetFrozen
	ClearBudgetFrozen                   = cp.ClearBudgetFrozen
	RewarmCampaignBudgetKeys            = cp.RewarmCampaignBudgetKeys
	EnableSlotMigrationDualWrite        = cp.EnableSlotMigrationDualWrite
	DisableSlotMigrationDualWrite       = cp.DisableSlotMigrationDualWrite
	CatchUpSlotMigrationDeltas          = cp.CatchUpSlotMigrationDeltas
	SlotMigrationReplicationLag         = cp.SlotMigrationReplicationLag
	PublishSlotMigrationDeltaTestHelper = cp.PublishSlotMigrationDeltaTestHelper
	ApplyCTVSettlement                  = cp.ApplyCTVSettlement
	PublishCampaignUpdateBroker         = cp.PublishCampaignUpdateBroker
	PublishSlotMapReload                = cp.PublishSlotMapReload
	IsRegistryFullSyncPayload           = cp.IsRegistryFullSyncPayload
	HashUserID                          = cp.HashUserID
	HashUserIDHex                       = cp.HashUserIDHex
	ConsentFlagsFromPurposes            = cp.ConsentFlagsFromPurposes
	NewConsentStore                     = cp.NewConsentStore
	NewCampaignRedisKeyCatalog          = cp.NewCampaignRedisKeyCatalog
	EncodeSlotMapReloadMessage          = cp.EncodeSlotMapReloadMessage
	DecodeSlotMapReloadMessage          = cp.DecodeSlotMapReloadMessage
	MigrationFenceRedisKey              = cp.MigrationFenceRedisKey
	BudgetFrozenRedisKey                = cp.BudgetFrozenRedisKey
	NewBudgetDeltaAggregator            = cp.NewBudgetDeltaAggregator
	ReloadRtbDeals                      = cp.ReloadRtbDeals
	RtbCatalogReloadChannel             = cp.RtbCatalogReloadChannel
	BudgetCampaignKey                   = cp.BudgetCampaignKey
)

var (
	ErrQuotaBudgetExceeded           = cp.ErrQuotaBudgetExceeded
	ErrQuotaInvalidChunk             = cp.ErrQuotaInvalidChunk
	ErrInsufficientCustomerBalance   = cp.ErrInsufficientCustomerBalance
	ErrCampaignSpendSkipped          = cp.ErrCampaignSpendSkipped
	ErrSlotMapIncomplete             = cp.ErrSlotMapIncomplete
	ErrSlotMapVersionNotFound        = cp.ErrSlotMapVersionNotFound
	ErrSlotMapInvalidSlot            = cp.ErrSlotMapInvalidSlot
	ErrSlotMapInvalidShard           = cp.ErrSlotMapInvalidShard
	ErrSlotMapAlreadyActive          = cp.ErrSlotMapAlreadyActive
	ErrTCPControlCorrupt             = cp.ErrTCPControlCorrupt
	ErrTCPControlHMAC                = cp.ErrTCPControlHMAC
	ErrCTVSettlementCampaignNotFound = cp.ErrCTVSettlementCampaignNotFound
)

var (
	DecodeUDPHeader                 = cp.DecodeUDPHeader
	DecodeUDPConfigRequest          = cp.DecodeUDPConfigRequest
	ComputeUDPConfigHash            = cp.ComputeUDPConfigHash
	ComputeUDPConfigHashWithWeights = cp.ComputeUDPConfigHashWithWeights
	NodeWeightsDrainFrozen          = cp.NodeWeightsDrainFrozen
	EffectiveNodeWeights            = cp.EffectiveNodeWeights
	EncodeTCPAckPayload             = cp.EncodeTCPAckPayload
	ProvenanceToUDPCode             = cp.ProvenanceToUDPCode
	DecodeTCPControlFrame           = cp.DecodeTCPControlFrame
	EncodeTCPControlFrame           = cp.EncodeTCPControlFrame
	EncodeTCPLimitsPayload          = cp.EncodeTCPLimitsPayload
	DecodeTCPAckPayload             = cp.DecodeTCPAckPayload
)

var (
	NewRegistry                          = cp.NewRegistry
	NewFlowRouter                        = cp.NewFlowRouter
	BanditSelect                         = cp.BanditSelect
	SelectSnapshot                       = cp.SelectSnapshot
	NewSettingsWatcher                   = cp.NewSettingsWatcher
	NewDCASNTable                        = cp.NewDCASNTable
	NewMobileCarrierASNTable             = cp.NewMobileCarrierASNTable
	NewLatencyRing                       = cp.NewLatencyRing
	NewCIDRTable                         = cp.NewCIDRTable
	BuildCIDRTableFromPrefixes           = cp.BuildCIDRTableFromPrefixes
	NewStreamProducer                    = cp.NewStreamProducer
	NewStreamProducerQueueForTest        = cp.NewStreamProducerQueueForTest
	NewStreamConsumer                    = cp.NewStreamConsumer
	NewFraudStreamWriter                 = cp.NewFraudStreamWriter
	NewFraudStreamWriterNearFullForTest  = cp.NewFraudStreamWriterNearFullForTest
	ReadFraudAggForce                    = cp.ReadFraudAggForce
	FraudAggForceKey                     = cp.FraudAggForceKey
	NewLocalQuantaLedger                 = cp.NewLocalQuantaLedger
	NewLocalQuantaStrict                 = cp.NewLocalQuantaStrict
	NewQuotaRefillWorker                 = cp.NewQuotaRefillWorker
	NewBudgetDeltaPublisher              = cp.NewBudgetDeltaPublisher
	NewLocalQuantaStreamPublisher        = cp.NewLocalQuantaStreamPublisher
	NewLocalQuantaStreamPublisherForTest = cp.NewLocalQuantaStreamPublisherForTest
	NewLocalQuotaCache                   = cp.NewLocalQuotaCache
	NewResidentialProxyRing              = cp.NewResidentialProxyRing
	DefaultResidentialProxyPolicyForTest = cp.DefaultResidentialProxyPolicyForTest
	ResidentialProxySignalForTest        = cp.ResidentialProxySignalForTest
	NewResidentialIntelTable             = cp.NewResidentialIntelTable
	NewProxyVPNTable                     = cp.NewProxyVPNTable
	NewStaticSlotSharder                 = cp.NewStaticSlotSharder
	NewLocalClickIdemCache               = cp.NewLocalClickIdemCache
	NewBrokerProducerSet                 = cp.NewBrokerProducerSet
	NewJumpHashSharder                   = cp.NewJumpHashSharder
	MigrationFenceKeyPrefix              = cp.MigrationFenceKeyPrefix
	SetStoreRetryPolicy                  = cp.SetStoreRetryPolicy
	NewBudgetCacheWarmer                 = cp.NewBudgetCacheWarmer
	NewClickHouseStore                   = cp.NewClickHouseStore
	NewPostgresStore                     = cp.NewPostgresStore
	NewPostgresStoreWithGate             = cp.NewPostgresStoreWithGate
	OpenClickHouseSpool                  = cp.OpenClickHouseSpool
	OpenClickHouseSpoolWithConfig        = cp.OpenClickHouseSpoolWithConfig
	DefaultClickHouseSpoolConfig         = cp.DefaultClickHouseSpoolConfig
	MarshalCHSpoolPayload                = cp.MarshalCHSpoolPayload
	NewMaxMindProvider                   = cp.NewMaxMindProvider
	NewGeoIPWatcher                      = cp.NewGeoIPWatcher
	NewDCASNFeedLoader                   = cp.NewDCASNFeedLoader
	ParseMobileCarrierASNs               = cp.ParseMobileCarrierASNs
	ResidentialProxyPolicyFromEnv        = cp.ResidentialProxyPolicyFromEnv
	NewConsentFilter                     = cp.NewConsentFilter
	FilterRedisOptions                   = cp.FilterRedisOptions
	FilterRedisReadTimeoutMs             = cp.FilterRedisReadTimeoutMs
	NewLocalQuantaFlusher                = cp.NewLocalQuantaFlusher
	FetchRecoveryDeltas                  = cp.FetchRecoveryDeltas
	SetRegistryQuantaFlushHook           = cp.SetRegistryQuantaFlushHook
	InvokeRegistryQuantaFlush            = cp.InvokeRegistryQuantaFlush
	LocalQuotaReturnScript               = cp.LocalQuotaReturnScript
	LocalQuantaStreamUsable              = cp.LocalQuantaStreamUsable
	SettingsPGSync                       = cp.SettingsPGSync
	NewLicenseFilter                     = cp.NewLicenseFilter
	NewVPPFilter                         = cp.NewVPPFilter
	NewHybridBalancer                    = cp.NewHybridBalancer
	DefaultBrokerProducerConfig          = cp.DefaultBrokerProducerConfig
	NewBrokerProducer                    = cp.NewBrokerProducer
	AdaptiveChunkSize                    = cp.AdaptiveChunkSize
	AdaptiveChunkSizeStrict              = cp.AdaptiveChunkSizeStrict
	NewFraudBrokerSink                   = cp.NewFraudBrokerSink
	StartFraudBackpressureWatcher        = cp.StartFraudBackpressureWatcher
	NewCIDRFeedLoader                    = cp.NewCIDRFeedLoader
	NewProxyVPNFeedLoader                = cp.NewProxyVPNFeedLoader
	NewModeratorIPTable                  = cp.NewModeratorIPTable
	NewModeratorIntelFeedLoader          = cp.NewModeratorIntelFeedLoader
	NewCampaignFlowTable                 = cp.NewCampaignFlowTable
	NewCampaignFlowRegistrySnapshot      = cp.NewCampaignFlowRegistrySnapshot
	NewCampaignFlowSync                  = cp.NewCampaignFlowSync
	ModeratorIntelFeedFileName           = cp.ModeratorIntelFeedFileName
	ModeratorIntelSigFileName            = cp.ModeratorIntelSigFileName
)

func FraudSignalWeight(id FraudReasonID) uint8 { return cp.FraudSignalWeight(id) }
func FraudSignalFlags(id FraudReasonID) uint8  { return cp.FraudSignalFlags(id) }
func ParseProxyVPNFeedLine(line string) (netip.Prefix, uint8, uint32, bool) {
	return cp.ParseProxyVPNFeedLine(line)
}
func ProxyVPNConnTypeBlocks(connType uint8) bool       { return cp.ProxyVPNConnTypeBlocks(connType) }
func HashResidentialProxyUser(s string) uint32         { return cp.HashResidentialProxyUser(s) }
func HashResidentialProxyUA(s string) uint32           { return cp.HashResidentialProxyUA(s) }
func RemainingBudgetMicro(camp *domain.Campaign) int64 { return cp.RemainingBudgetMicro(camp) }
func ToUUID(u uuid.UUID) pgtype.UUID                   { return cp.ToUUID(u) }
func BuildIngressQuotaMap(epoch int64, limits *UDPControlLimits, numWorkers int) *IngressQuotaMap {
	return cp.BuildIngressQuotaMap(epoch, limits, numWorkers)
}
func FraudReasonCode(id FraudReasonID) string   { return cp.FraudReasonCode(id) }
func CachedTimeIn(loc *time.Location) time.Time { return cp.CachedTimeIn(loc) }
func CachedTimeUTC() time.Time                  { return cp.CachedTimeUTC() }
func IngressDayKey(buf []byte, regionCode uint8, customerID uuid.UUID, dateStr string) []byte {
	return cp.IngressDayKey(buf, regionCode, customerID, dateStr)
}
func ShouldSampleHistogram(seq uint64, mask uint64) bool { return cp.ShouldSampleHistogram(seq, mask) }

var (
	BuildDCASNSnapshot     = cp.BuildDCASNSnapshot
	ScanUAFamily           = cp.ScanUAFamily
	HashTCPSynFields       = cp.HashTCPSynFields
	MonoElapsedSeconds     = cp.MonoElapsedSeconds
	CampaignSyncKey        = cp.CampaignSyncKey
	CIDRFeedNames          = cp.CIDRFeedNames
	RedisClusterSlot       = cp.RedisClusterSlot
	WithStaticCampaign     = cp.WithStaticCampaign
	PublishTCPSynSigCorpus = cp.PublishTCPSynSigCorpus
)
