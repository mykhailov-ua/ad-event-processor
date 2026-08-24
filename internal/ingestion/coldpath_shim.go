package ingestion

import (
	"ad-event-processor/internal/domain"
)

type (
	CampaignRepo                 = domain.CampaignRepo
	CustomerRepo                 = domain.CustomerRepo
	QuotaRepo                    = domain.QuotaRepo
	SlotMapRepo                  = domain.SlotMapRepo
	SlotMigrationRepo            = domain.SlotMigrationRepo
	CampaignRoutingRepo          = domain.CampaignRoutingRepo
	SyncWorker                   = domain.SyncWorker
	SpendFlushItem               = domain.SpendFlushItem
	SpendFlushOutcome            = domain.SpendFlushOutcome
	PendingRollup                = domain.PendingRollup
	BudgetReconSnapshot          = domain.BudgetReconSnapshot
	BudgetInvariantSnapshot      = domain.BudgetInvariantSnapshot
	SlotOverride                 = domain.SlotOverride
	ReserveChunkResult           = domain.ReserveChunkResult
	MarginEconomicsSplit         = domain.MarginEconomicsSplit
	ConsentStore                 = domain.ConsentStore
	CampaignKeyMigrator          = domain.CampaignKeyMigrator
	CampaignRedisKeyCatalog      = domain.CampaignRedisKeyCatalog
	SlotMigrationDualWriteConfig = domain.SlotMigrationDualWriteConfig
	SlotMigrationDelta           = domain.SlotMigrationDelta
	OpsSlotMapResponse           = domain.OpsSlotMapResponse
	CTVSettlementResult          = domain.CTVSettlementResult
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
	MaxLedgerBatchSize                  = domain.MaxLedgerBatchSize
	BudgetLockTTLSeconds                = domain.BudgetLockTTLSeconds
	FetchBudgetReconSnapshot            = domain.FetchBudgetReconSnapshot
	ReadBudgetInvariant                 = domain.ReadBudgetInvariant
	ReadBudgetInvariants                = domain.ReadBudgetInvariants
	VerifyBudgetInvariant               = domain.VerifyBudgetInvariant
	AssertBudgetInvariant               = domain.AssertBudgetInvariant
	CampaignShardID                     = domain.CampaignShardID
	CampaignSlotIndex                   = domain.CampaignSlotIndex
	FilterCampaignIDsBySlot             = domain.FilterCampaignIDsBySlot
	ComputeMarginEconomicsSplit         = domain.ComputeMarginEconomicsSplit
	WriteMarginEconomicsLegs            = domain.WriteMarginEconomicsLegs
	TableFromRows                       = domain.TableFromRows
	CampaignFromDBRow                   = domain.CampaignFromDBRow
	CampaignFromGetCampaignFullRow      = domain.CampaignFromGetCampaignFullRow
	CampaignFromListActiveCampaignsRow  = domain.CampaignFromListActiveCampaignsRow
	EncodeQuotaEpochDatagramWithWeights = domain.EncodeQuotaEpochDatagramWithWeights
	MarshalEpochPayload                 = domain.MarshalEpochPayload
	NodeWeightsToJSON                   = domain.NodeWeightsToJSON
	ControlFailOpenEnabled              = domain.ControlFailOpenEnabled
	EdgeControlEqualizeWeights          = domain.EdgeControlEqualizeWeights
	EdgeControlDrainFrozen              = domain.EdgeControlDrainFrozen
	QuotaShardForCampaign               = domain.QuotaShardForCampaign
	EqualizeNodeWeights                 = domain.EqualizeNodeWeights
	LoadActiveSlotMap                   = domain.LoadActiveSlotMap
	ReloadStaticSlotMapIfChanged        = domain.ReloadStaticSlotMapIfChanged
	SlotMapShardTable                   = domain.SlotMapShardTable
	BumpMigrationFences                 = domain.BumpMigrationFences
	SetBudgetFrozen                     = domain.SetBudgetFrozen
	ClearBudgetFrozen                   = domain.ClearBudgetFrozen
	RewarmCampaignBudgetKeys            = domain.RewarmCampaignBudgetKeys
	EnableSlotMigrationDualWrite        = domain.EnableSlotMigrationDualWrite
	DisableSlotMigrationDualWrite       = domain.DisableSlotMigrationDualWrite
	CatchUpSlotMigrationDeltas          = domain.CatchUpSlotMigrationDeltas
	SlotMigrationReplicationLag         = domain.SlotMigrationReplicationLag
	PublishSlotMigrationDeltaTestHelper = domain.PublishSlotMigrationDeltaTestHelper
	ApplyCTVSettlement                  = domain.ApplyCTVSettlement
	PublishCampaignUpdateBroker         = domain.PublishCampaignUpdateBroker
	PublishSlotMapReload                = domain.PublishSlotMapReload
	IsRegistryFullSyncPayload           = domain.IsRegistryFullSyncPayload
	HashUserID                          = domain.HashUserID
	HashUserIDHex                       = domain.HashUserIDHex
	ConsentFlagsFromPurposes            = domain.ConsentFlagsFromPurposes
	NewConsentStore                     = domain.NewConsentStore
	NewCampaignRedisKeyCatalog          = domain.NewCampaignRedisKeyCatalog
	EncodeSlotMapReloadMessage          = domain.EncodeSlotMapReloadMessage
	DecodeSlotMapReloadMessage          = domain.DecodeSlotMapReloadMessage
	MigrationFenceRedisKey              = domain.MigrationFenceRedisKey
	BudgetFrozenRedisKey                = domain.BudgetFrozenRedisKey
	NewBudgetDeltaAggregator            = domain.NewBudgetDeltaAggregator
	ReloadRtbDeals                      = domain.ReloadRtbDeals
	RtbCatalogReloadChannel             = domain.RtbCatalogReloadChannel
)

const (
	SlotCount                        = domain.SlotCount
	SlotMask                         = domain.SlotMask
	ConsentPurposeAdStorage          = domain.ConsentPurposeAdStorage
	ConsentPurposeAnalytics          = domain.ConsentPurposeAnalytics
	ConsentRedisKeyPrefix            = domain.ConsentRedisKeyPrefix
	ConsentDefaultUpdateChannel      = domain.ConsentDefaultUpdateChannel
	DefaultCampaignUpdateBrokerTopic = domain.DefaultCampaignUpdateBrokerTopic
	RegistryFullSyncPayload          = domain.RegistryFullSyncPayload
	DefaultSlotMapReloadTopic        = domain.DefaultSlotMapReloadTopic
	SlotMigrationDualWriteFlagKey    = domain.SlotMigrationDualWriteFlagKey
	DefaultRtbCatalogReloadChannel   = domain.DefaultRtbCatalogReloadChannel
)

var (
	ErrQuotaBudgetExceeded           = domain.ErrQuotaBudgetExceeded
	ErrQuotaInvalidChunk             = domain.ErrQuotaInvalidChunk
	ErrInsufficientCustomerBalance   = domain.ErrInsufficientCustomerBalance
	ErrCampaignSpendSkipped          = domain.ErrCampaignSpendSkipped
	ErrSlotMapIncomplete             = domain.ErrSlotMapIncomplete
	ErrSlotMapVersionNotFound        = domain.ErrSlotMapVersionNotFound
	ErrSlotMapInvalidSlot            = domain.ErrSlotMapInvalidSlot
	ErrSlotMapInvalidShard           = domain.ErrSlotMapInvalidShard
	ErrSlotMapAlreadyActive          = domain.ErrSlotMapAlreadyActive
	ErrTCPControlCorrupt             = domain.ErrTCPControlCorrupt
	ErrTCPControlHMAC                = domain.ErrTCPControlHMAC
	ErrCTVSettlementCampaignNotFound = domain.ErrCTVSettlementCampaignNotFound
)

type (
	UDPControlLimits        = domain.UDPControlLimits
	UDPHeader               = domain.UDPHeader
	UDPNodeWeight           = domain.UDPNodeWeight
	UDPConfigRequestPayload = domain.UDPConfigRequestPayload
	UDPNodeWeightJSON       = domain.UDPNodeWeightJSON
	TCPControlHeader        = domain.TCPControlHeader
	TCPAckPayload           = domain.TCPAckPayload
)

const (
	UDPHeaderSize          = domain.UDPHeaderSize
	UDPMaxControlShards    = domain.UDPMaxControlShards
	UDPMsgQuotaEpoch       = domain.UDPMsgQuotaEpoch
	UDPMsgConfigSnapshot   = domain.UDPMsgConfigSnapshot
	UDPMsgConfigRequest    = domain.UDPMsgConfigRequest
	UDPMsgMigrationBarrier = domain.UDPMsgMigrationBarrier
	UDPFlagSnapshot        = domain.UDPFlagSnapshot
	UDPFlagNodeWeights     = domain.UDPFlagNodeWeights
	TCPControlHeaderSize   = domain.TCPControlHeaderSize
	TCPMsgSnapshot         = domain.TCPMsgSnapshot
	TCPMsgSnapshotRequest  = domain.TCPMsgSnapshotRequest
	TCPMsgAck              = domain.TCPMsgAck
)

var (
	DecodeUDPHeader                 = domain.DecodeUDPHeader
	DecodeUDPConfigRequest          = domain.DecodeUDPConfigRequest
	ComputeUDPConfigHash            = domain.ComputeUDPConfigHash
	ComputeUDPConfigHashWithWeights = domain.ComputeUDPConfigHashWithWeights
	NodeWeightsDrainFrozen          = domain.NodeWeightsDrainFrozen
	EffectiveNodeWeights            = domain.EffectiveNodeWeights
	EncodeTCPAckPayload             = domain.EncodeTCPAckPayload
	ProvenanceToUDPCode             = domain.ProvenanceToUDPCode
	DecodeTCPControlFrame           = domain.DecodeTCPControlFrame
	EncodeTCPControlFrame           = domain.EncodeTCPControlFrame
	EncodeTCPLimitsPayload          = domain.EncodeTCPLimitsPayload
	DecodeTCPAckPayload             = domain.DecodeTCPAckPayload
)
