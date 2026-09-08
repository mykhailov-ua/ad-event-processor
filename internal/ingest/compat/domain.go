package compat

import (
	"net/netip"
	"sync/atomic"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/budget"
	"ad-event-processor/internal/domain/shard"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/internal/stream"
	streamauditlog "ad-event-processor/internal/stream/auditlog"
	streamfraud "ad-event-processor/internal/stream/fraud"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	FraudReasonID = filter.FraudReasonID
)

const (
	FraudReasonNone                     = filter.FraudReasonNone
	FraudReasonDatacenterIP             = filter.FraudReasonDatacenterIP
	FraudReasonLowTTC                   = filter.FraudReasonLowTTC
	FraudReasonMissingImpTS             = filter.FraudReasonMissingImpTS
	FraudReasonL3Blocklist              = filter.FraudReasonL3Blocklist
	FraudReasonTLSBlocklist             = filter.FraudReasonTLSBlocklist
	FraudReasonDeviceMismatch           = filter.FraudReasonDeviceMismatch
	FraudReasonTCPMSSAnomaly            = filter.FraudReasonTCPMSSAnomaly
	FraudReasonTCPTunnelMSS             = filter.FraudReasonTCPTunnelMSS
	FraudReasonTCPSynOSMismatch         = filter.FraudReasonTCPSynOSMismatch
	FraudReasonJSONSerializationBot     = filter.FraudReasonJSONSerializationBot
	FraudReasonOSFingerprint            = filter.FraudReasonOSFingerprint
	FraudReasonIPv4Rotation             = filter.FraudReasonIPv4Rotation
	FraudReasonResidentialProxy         = filter.FraudReasonResidentialProxy
	FraudReasonAttestationMissing       = filter.FraudReasonAttestationMissing
	FraudReasonModeratorIP              = filter.FraudReasonModeratorIP
	FraudReasonSecFetchAnomaly          = filter.FraudReasonSecFetchAnomaly
	FraudReasonClientHintsMismatch      = filter.FraudReasonClientHintsMismatch
	FraudReasonTLSALPNMismatch          = filter.FraudReasonTLSALPNMismatch
	FraudReasonH2SettingsMismatch       = filter.FraudReasonH2SettingsMismatch
	FraudReasonH2PseudoOrder            = filter.FraudReasonH2PseudoOrder
	FraudReasonH2DowngradeArtifact      = filter.FraudReasonH2DowngradeArtifact
	FraudReasonHeaderOrderMismatch      = filter.FraudReasonHeaderOrderMismatch
	FraudReasonAcceptEncodingMismatch   = filter.FraudReasonAcceptEncodingMismatch
	FraudReasonAcceptLangGeoMismatch    = filter.FraudReasonAcceptLangGeoMismatch
	FraudReasonTLSJA4Mismatch           = filter.FraudReasonTLSJA4Mismatch
	FraudReasonBehaviorTelemetryMissing = filter.FraudReasonBehaviorTelemetryMissing
	FraudReasonBehaviorBezierBot        = filter.FraudReasonBehaviorBezierBot
	fraudReasonCount                    = filter.FraudReasonID(filter.FraudReasonCount)
)

func FraudSignalWeight(id FraudReasonID) uint8 {
	return filter.FraudSignalWeight(id)
}

func FraudSignalFlags(id FraudReasonID) uint8 {
	return filter.FraudSignalFlags(id)
}

const (
	ProxyVPNConnISP     = filter.ProxyVPNConnISP
	ProxyVPNConnHosting = filter.ProxyVPNConnHosting
	ProxyVPNConnVPN     = filter.ProxyVPNConnVPN
	ProxyVPNConnMobile  = filter.ProxyVPNConnMobile
)

func ParseProxyVPNFeedLine(line string) (netip.Prefix, uint8, uint32, bool) {
	return filter.ParseProxyVPNFeedLine(line)
}

func ProxyVPNConnTypeBlocks(connType uint8) bool {
	return filter.ProxyVPNConnTypeBlocks(connType)
}

var BuildDCASNSnapshot = filter.BuildDCASNSnapshot

func HashResidentialProxyUser(s string) uint32 {
	return filter.HashResidentialProxyUser(s)
}

func HashResidentialProxyUA(s string) uint32 {
	return filter.HashResidentialProxyUA(s)
}

func RemainingBudgetMicro(camp *domain.Campaign) int64 {
	return filter.RemainingBudgetMicro(camp)
}

const (
	uaFamilyWindows = filter.UAFamilyWindows
	uaFamilyMac     = filter.UAFamilyMac
	uaFamilyLinux   = filter.UAFamilyLinux
	uaFamilyMobile  = filter.UAFamilyMobile
	UAFamilyWindows = filter.UAFamilyWindows
	UAFamilyMac     = filter.UAFamilyMac
	UAFamilyLinux   = filter.UAFamilyLinux
	UAFamilyMobile  = filter.UAFamilyMobile
	UAFamilyUnknown = filter.UAFamilyUnknown
)

var (
	ErrSegmentNotIncluded = filter.ErrSegmentNotIncluded
	ErrSegmentExcluded    = filter.ErrSegmentExcluded
)

var (
	ErrConsentDenied                = filter.ErrConsentDenied
	FraudReasonCodeDatacenterIP     = filter.FraudReasonCodeDatacenterIP
	FraudReasonCodeL3Blocklist      = filter.FraudReasonCodeL3Blocklist
	FraudReasonCodeIPv4Rotation     = filter.FraudReasonCodeIPv4Rotation
	FraudReasonCodeTCPSynOSMismatch = filter.FraudReasonCodeTCPSynOSMismatch
	ClassifyFilterErr               = filter.ClassifyFilterErr
	ParseASNLine                    = filter.ParseASNLine
)

func WithStaticCampaign(fn func(camp **domain.Campaign)) {
	filter.WithStaticCampaign(fn)
}

var (
	CRC32Castagnoli       = filter.CRC32Castagnoli
	AppendCampaignHashTag = filter.AppendCampaignHashTag
	PlacementBlacklistKey = filter.PlacementBlacklistKey
	ScanUAFamily          = filter.ScanUAFamily
)

func WriteAuditLog(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
) {
	streamauditlog.Write(l, seq, sampleMask, shardID, evt)
}

func EnqueueFraudReject(writer *streamfraud.FraudStreamWriter, shard int, evt *domain.Event) {
	streamfraud.EnqueueFraudReject(writer, shard, evt)
}

const (
	FraudReasonCodeMissingImpTS  = filter.FraudReasonCodeMissingImpTS
	FraudReasonCodeLowTTC        = filter.FraudReasonCodeLowTTC
	FraudReasonCodeOSFingerprint = filter.FraudReasonCodeOSFingerprint
)

func PublishTCPSynSigCorpus(snap *filter.TCPSynSigCorpusSnapshot) {
	filter.PublishTCPSynSigCorpus(snap)
}

type (
	CampaignRepo                 = domain.CampaignRepo
	CustomerRepo                 = domain.CustomerRepo
	QuotaRepo                    = domain.QuotaRepo
	SlotMapRepo                  = domain.SlotMapRepo
	SlotMigrationRepo            = domain.SlotMigrationRepo
	CampaignRoutingRepo          = domain.CampaignRoutingRepo
	SyncWorker                   = domain.SyncWorker
	SpendFlushItem               = budget.SpendFlushItem
	SpendFlushOutcome            = budget.SpendFlushOutcome
	PendingRollup                = budget.PendingRollup
	BudgetReconSnapshot          = budget.BudgetReconSnapshot
	BudgetInvariantSnapshot      = budget.BudgetInvariantSnapshot
	SlotOverride                 = domain.SlotOverride
	ReserveChunkResult           = domain.ReserveChunkResult
	MarginEconomicsSplit         = domain.MarginEconomicsSplit
	ConsentStore                 = domain.ConsentStore
	CampaignKeyMigrator          = domain.CampaignKeyMigrator
	CampaignRedisKeyCatalog      = domain.CampaignRedisKeyCatalog
	SlotMigrationDualWriteConfig = domain.SlotMigrationDualWriteConfig
	SlotMigrationDelta           = domain.SlotMigrationDelta
	OpsSlotMapResponse           = domain.OpsSlotMapResponse
	CTVSettlementResult          = budget.CTVSettlementResult
	BudgetDeltaAggregator        = domain.BudgetDeltaAggregator
	RtbBidShadeInput             = domain.RtbBidShadeInput
	RtbBidShadeOutput            = domain.RtbBidShadeOutput
)

var (
	NewCampaignRepo                     = domain.NewCampaignRepo
	NewCampaignRepoWithDB               = domain.NewCampaignRepoWithDB
	NewCustomerRepo                     = domain.NewCustomerRepo
	NewCustomerRepoWithDB               = domain.NewCustomerRepoWithDB
	NewQuotaRepo                        = domain.NewQuotaRepo
	NewSlotMapRepo                      = domain.NewSlotMapRepo
	NewSlotMigrationRepo                = domain.NewSlotMigrationRepo
	NewCampaignRoutingRepo              = domain.NewCampaignRoutingRepo
	NewSyncWorker                       = domain.NewSyncWorker
	MaxLedgerBatchSize                  = budget.MaxLedgerBatchSize
	BudgetLockTTLSeconds                = domain.BudgetLockTTLSeconds
	FetchBudgetReconSnapshot            = budget.FetchBudgetReconSnapshot
	ReadBudgetInvariant                 = budget.ReadBudgetInvariant
	ReadBudgetInvariants                = budget.ReadBudgetInvariants
	VerifyBudgetInvariant               = budget.VerifyBudgetInvariant
	AssertBudgetInvariant               = budget.AssertBudgetInvariant
	CampaignShardID                     = domain.CampaignShardID
	CampaignSlotIndex                   = shard.CampaignSlotIndex
	FilterCampaignIDsBySlot             = shard.FilterCampaignIDsBySlot
	ComputeMarginEconomicsSplit         = domain.ComputeMarginEconomicsSplit
	WriteMarginEconomicsLegs            = domain.WriteMarginEconomicsLegs
	TableFromRows                       = shard.TableFromRows
	CampaignFromDBRow                   = domain.CampaignFromDBRow
	CampaignFromGetCampaignFullRow      = domain.CampaignFromGetCampaignFullRow
	CampaignFromListActiveCampaignsRow  = domain.CampaignFromListActiveCampaignsRow
	EncodeQuotaEpochDatagramWithWeights = shard.EncodeQuotaEpochDatagramWithWeights
	MarshalEpochPayload                 = shard.MarshalEpochPayload
	NodeWeightsToJSON                   = shard.NodeWeightsToJSON
	ControlFailOpenEnabled              = shard.ControlFailOpenEnabled
	EdgeControlEqualizeWeights          = shard.EdgeControlEqualizeWeights
	EdgeControlDrainFrozen              = shard.EdgeControlDrainFrozen
	QuotaShardForCampaign               = domain.QuotaShardForCampaign
	EqualizeNodeWeights                 = shard.EqualizeNodeWeights
	LoadActiveSlotMap                   = shard.LoadActiveSlotMap
	ReloadStaticSlotMapIfChanged        = shard.ReloadStaticSlotMapIfChanged
	SlotMapShardTable                   = shard.SlotMapShardTable
	BumpMigrationFences                 = shard.BumpMigrationFences
	SetBudgetFrozen                     = shard.SetBudgetFrozen
	ClearBudgetFrozen                   = shard.ClearBudgetFrozen
	RewarmCampaignBudgetKeys            = shard.RewarmCampaignBudgetKeys
	EnableSlotMigrationDualWrite        = shard.EnableSlotMigrationDualWrite
	DisableSlotMigrationDualWrite       = shard.DisableSlotMigrationDualWrite
	CatchUpSlotMigrationDeltas          = shard.CatchUpSlotMigrationDeltas
	SlotMigrationReplicationLag         = shard.SlotMigrationReplicationLag
	PublishSlotMigrationDeltaTestHelper = shard.PublishSlotMigrationDeltaTestHelper
	ApplyCTVSettlement                  = budget.ApplyCTVSettlement
	PublishCampaignUpdateBroker         = budget.PublishCampaignUpdateBroker
	PublishSlotMapReload                = shard.PublishSlotMapReload
	IsRegistryFullSyncPayload           = budget.IsRegistryFullSyncPayload
	HashUserID                          = domain.HashUserID
	HashUserIDHex                       = domain.HashUserIDHex
	ConsentFlagsFromPurposes            = domain.ConsentFlagsFromPurposes
	NewConsentStore                     = domain.NewConsentStore
	NewCampaignRedisKeyCatalog          = domain.NewCampaignRedisKeyCatalog
	EncodeSlotMapReloadMessage          = shard.EncodeSlotMapReloadMessage
	DecodeSlotMapReloadMessage          = shard.DecodeSlotMapReloadMessage
	MigrationFenceRedisKey              = shard.MigrationFenceRedisKey
	BudgetFrozenRedisKey                = shard.BudgetFrozenRedisKey
	NewBudgetDeltaAggregator            = domain.NewBudgetDeltaAggregator
	ReloadRtbDeals                      = rtb.ReloadDeals
	RtbCatalogReloadChannel             = domain.RtbCatalogReloadChannel
	BudgetCampaignKey                   = domain.BudgetCampaignKey
	CampaignSyncKey                     = domain.CampaignSyncKey
	RedisClusterSlot                    = domain.RedisClusterSlot
)

const (
	SlotCount                        = shard.SlotCount
	SlotMask                         = shard.SlotMask
	ConsentPurposeAdStorage          = domain.ConsentPurposeAdStorage
	ConsentPurposeAnalytics          = domain.ConsentPurposeAnalytics
	ConsentRedisKeyPrefix            = domain.ConsentRedisKeyPrefix
	ConsentDefaultUpdateChannel      = domain.ConsentDefaultUpdateChannel
	DefaultCampaignUpdateBrokerTopic = budget.DefaultCampaignUpdateBrokerTopic
	RegistryFullSyncPayload          = budget.RegistryFullSyncPayload
	DefaultSlotMapReloadTopic        = shard.DefaultSlotMapReloadTopic
	SlotMigrationDualWriteFlagKey    = shard.SlotMigrationDualWriteFlagKey
	DefaultRtbCatalogReloadChannel   = domain.DefaultRtbCatalogReloadChannel
)

var (
	ErrQuotaBudgetExceeded           = domain.ErrQuotaBudgetExceeded
	ErrQuotaInvalidChunk             = domain.ErrQuotaInvalidChunk
	ErrInsufficientCustomerBalance   = budget.ErrInsufficientCustomerBalance
	ErrCampaignSpendSkipped          = budget.ErrCampaignSpendSkipped
	ErrSlotMapIncomplete             = shard.ErrSlotMapIncomplete
	ErrSlotMapVersionNotFound        = shard.ErrSlotMapVersionNotFound
	ErrSlotMapInvalidSlot            = shard.ErrSlotMapInvalidSlot
	ErrSlotMapInvalidShard           = shard.ErrSlotMapInvalidShard
	ErrSlotMapAlreadyActive          = shard.ErrSlotMapAlreadyActive
	ErrTCPControlCorrupt             = domain.ErrTCPControlCorrupt
	ErrTCPControlHMAC                = domain.ErrTCPControlHMAC
	ErrCTVSettlementCampaignNotFound = budget.ErrCTVSettlementCampaignNotFound
)

type (
	UDPControlLimits        = shard.UDPControlLimits
	UDPHeader               = shard.UDPHeader
	UDPNodeWeight           = shard.UDPNodeWeight
	UDPConfigRequestPayload = shard.UDPConfigRequestPayload
	UDPNodeWeightJSON       = shard.UDPNodeWeightJSON
	TCPControlHeader        = domain.TCPControlHeader
	TCPAckPayload           = domain.TCPAckPayload
)

const (
	UDPHeaderSize          = shard.UDPHeaderSize
	UDPMaxControlShards    = shard.UDPMaxControlShards
	UDPMsgQuotaEpoch       = shard.UDPMsgQuotaEpoch
	UDPMsgConfigSnapshot   = shard.UDPMsgConfigSnapshot
	UDPMsgConfigRequest    = shard.UDPMsgConfigRequest
	UDPMsgMigrationBarrier = shard.UDPMsgMigrationBarrier
	UDPFlagSnapshot        = shard.UDPFlagSnapshot
	UDPFlagNodeWeights     = shard.UDPFlagNodeWeights
	TCPControlHeaderSize   = domain.TCPControlHeaderSize
	TCPMsgSnapshot         = domain.TCPMsgSnapshot
	TCPMsgSnapshotRequest  = domain.TCPMsgSnapshotRequest
	TCPMsgAck              = domain.TCPMsgAck
)

var (
	DecodeUDPHeader                 = shard.DecodeUDPHeader
	DecodeUDPConfigRequest          = shard.DecodeUDPConfigRequest
	ComputeUDPConfigHash            = shard.ComputeUDPConfigHash
	ComputeUDPConfigHashWithWeights = shard.ComputeUDPConfigHashWithWeights
	NodeWeightsDrainFrozen          = shard.NodeWeightsDrainFrozen
	EffectiveNodeWeights            = shard.EffectiveNodeWeights
	EncodeTCPAckPayload             = domain.EncodeTCPAckPayload
	ProvenanceToUDPCode             = shard.ProvenanceToUDPCode
	DecodeTCPControlFrame           = domain.DecodeTCPControlFrame
	EncodeTCPControlFrame           = domain.EncodeTCPControlFrame
	EncodeTCPLimitsPayload          = domain.EncodeTCPLimitsPayload
	DecodeTCPAckPayload             = domain.DecodeTCPAckPayload
)

const CampaignEpochKey = shard.CampaignEpochKey

const RtbFloorRedisKeyPrefix = domain.RtbFloorRedisKeyPrefix

func ToUUID(u uuid.UUID) pgtype.UUID {
	return shard.ToUUID(u)
}

type IngressQuotaMap = stream.IngressQuotaMap

func BuildIngressQuotaMap(epoch int64, limits *UDPControlLimits, numWorkers int) *IngressQuotaMap {
	return stream.BuildIngressQuotaMap(epoch, limits, numWorkers)
}
