package domain

import (
	budget "ad-event-processor/internal/domain/budget"
	shard "ad-event-processor/internal/domain/shard"
)

type (
	BudgetDeltaAggregator   = budget.BudgetDeltaAggregator
	BudgetInvariantSnapshot = budget.BudgetInvariantSnapshot
	BudgetManager           = budget.BudgetManager
	BudgetReconSnapshot     = budget.BudgetReconSnapshot
	CTVSettlementResult     = budget.CTVSettlementResult
	IngressCostConfig       = budget.IngressCostConfig
	IngressCostParam        = budget.IngressCostParam
	IngressCostPolicy       = budget.IngressCostPolicy
	MarginEconomicsSplit    = budget.MarginEconomicsSplit
	PaymentLedgerEntry      = budget.PaymentLedgerEntry
	PaymentSettlement       = budget.PaymentSettlement
	PendingRollup           = budget.PendingRollup
	SpendBatchFlusher       = budget.SpendBatchFlusher
	SpendFlushItem          = budget.SpendFlushItem
	SpendFlushOutcome       = budget.SpendFlushOutcome
)

type (
	BrokerConsumerConfig         = shard.BrokerConsumerConfig
	CampaignRoutingRepo          = shard.CampaignRoutingRepo
	EpochPayloadJSON             = shard.EpochPayloadJSON
	JumpHashSharder              = shard.JumpHashSharder
	OpsSlotMapResponse           = shard.OpsSlotMapResponse
	Sharder                      = shard.Sharder
	SlotMapReloadMessage         = shard.SlotMapReloadMessage
	SlotMapRepo                  = shard.SlotMapRepo
	SlotMapSnapshot              = shard.SlotMapSnapshot
	SlotMigrationDelta           = shard.SlotMigrationDelta
	SlotMigrationDualWriteConfig = shard.SlotMigrationDualWriteConfig
	SlotMigrationRepo            = shard.SlotMigrationRepo
	SlotOverride                 = shard.SlotOverride
	SlotTable                    = shard.SlotTable
	StaticSlotSharder            = shard.StaticSlotSharder
	UDPConfigRequestPayload      = shard.UDPConfigRequestPayload
	UDPControlLimits             = shard.UDPControlLimits
	UDPHeader                    = shard.UDPHeader
	UDPMigrationBarrierPayload   = shard.UDPMigrationBarrierPayload
	UDPNodeWeight                = shard.UDPNodeWeight
	UDPNodeWeightJSON            = shard.UDPNodeWeightJSON
)

var (
	ApplyCTVSettlement                  = budget.ApplyCTVSettlement
	AssertBudgetInvariant               = budget.AssertBudgetInvariant
	BatchFetchBudgetReconSnapshots      = budget.BatchFetchBudgetReconSnapshots
	ComputeMarginEconomicsSplit         = budget.ComputeMarginEconomicsSplit
	DefaultCampaignUpdateBrokerTopic    = budget.DefaultCampaignUpdateBrokerTopic
	ErrCTVSettlementCampaignNotFound    = budget.ErrCTVSettlementCampaignNotFound
	ErrCampaignSpendSkipped             = budget.ErrCampaignSpendSkipped
	ErrInsufficientCustomerBalance      = budget.ErrInsufficientCustomerBalance
	ErrSettlementCustomerNotFound       = budget.ErrSettlementCustomerNotFound
	ErrSettlementTopupNotFound          = budget.ErrSettlementTopupNotFound
	FetchBudgetReconSnapshot            = budget.FetchBudgetReconSnapshot
	IngressCostDisabled                 = budget.IngressCostDisabled
	IngressCostParamBid                 = budget.IngressCostParamBid
	IngressCostParamCPC                 = budget.IngressCostParamCPC
	IngressCostParamCost                = budget.IngressCostParamCost
	IngressCostPolicyIgnore             = budget.IngressCostPolicyIgnore
	IngressCostPolicyReject             = budget.IngressCostPolicyReject
	IsRegistryFullSyncPayload           = budget.IsRegistryFullSyncPayload
	IsSettlementNotFound                = budget.IsSettlementNotFound
	MaxLedgerBatchSize                  = budget.MaxLedgerBatchSize
	NewBudgetDeltaAggregator            = budget.NewBudgetDeltaAggregator
	ParseIngressCostConfigJSON          = budget.ParseIngressCostConfigJSON
	PublishCampaignUpdateBroker         = budget.PublishCampaignUpdateBroker
	ReadBudgetInvariant                 = budget.ReadBudgetInvariant
	ReadBudgetInvariants                = budget.ReadBudgetInvariants
	RegistryFullSyncPayload             = budget.RegistryFullSyncPayload
	VerifyBudgetInvariant               = budget.VerifyBudgetInvariant
	WriteMarginEconomicsLegs            = budget.WriteMarginEconomicsLegs
	AppendCampaignHashTag               = shard.AppendCampaignHashTag
	AppendCampaignSubHashTag            = shard.AppendCampaignSubHashTag
	AppendUUID                          = shard.AppendUUID
	ApplySlotMapToSharder               = shard.ApplySlotMapToSharder
	BudgetCampaignKey                   = shard.BudgetCampaignKey
	BudgetFrozenKeyPrefix               = shard.BudgetFrozenKeyPrefix
	BudgetFrozenRedisKey                = shard.BudgetFrozenRedisKey
	BudgetKeyTTL                        = shard.BudgetKeyTTL
	BudgetQuotaKey                      = shard.BudgetQuotaKey
	BudgetQuotaKeySub                   = shard.BudgetQuotaKeySub
	BuildSlotTable                      = shard.BuildSlotTable
	BumpMigrationFences                 = shard.BumpMigrationFences
	CRC32Castagnoli                     = shard.CRC32Castagnoli
	CampaignEpochKey                    = shard.CampaignEpochKey
	CampaignHashTag                     = shard.CampaignHashTag
	CampaignSlotIndex                   = shard.CampaignSlotIndex
	CampaignSyncKey                     = shard.CampaignSyncKey
	CatchUpSlotMigrationDeltas          = shard.CatchUpSlotMigrationDeltas
	CheckSlotMapRoutingParity           = shard.CheckSlotMapRoutingParity
	ClearBudgetFrozen                   = shard.ClearBudgetFrozen
	CompareSlotMaps                     = shard.CompareSlotMaps
	ComputeUDPConfigHash                = shard.ComputeUDPConfigHash
	ComputeUDPConfigHashWithWeights     = shard.ComputeUDPConfigHashWithWeights
	ControlFailOpenEnabled              = shard.ControlFailOpenEnabled
	CustomerSyncKey                     = shard.CustomerSyncKey
	DailySpendKeyPrefix                 = shard.DailySpendKeyPrefix
	DecodeSlotMapReloadMessage          = shard.DecodeSlotMapReloadMessage
	DecodeUDPConfigRequest              = shard.DecodeUDPConfigRequest
	DecodeUDPHeader                     = shard.DecodeUDPHeader
	DefaultSlotMapParitySamples         = shard.DefaultSlotMapParitySamples
	DefaultSlotMapReloadTopic           = shard.DefaultSlotMapReloadTopic
	DisableSlotMigrationDualWrite       = shard.DisableSlotMigrationDualWrite
	EdgeControlDrainFrozen              = shard.EdgeControlDrainFrozen
	EdgeControlEqualizeWeights          = shard.EdgeControlEqualizeWeights
	EffectiveNodeWeights                = shard.EffectiveNodeWeights
	EnableSlotMigrationDualWrite        = shard.EnableSlotMigrationDualWrite
	EncodeQuotaEpochDatagram            = shard.EncodeQuotaEpochDatagram
	EncodeQuotaEpochDatagramWithWeights = shard.EncodeQuotaEpochDatagramWithWeights
	EncodeSlotMapReloadMessage          = shard.EncodeSlotMapReloadMessage
	EncodeSlotMapReloadMessageVersion   = shard.EncodeSlotMapReloadMessageVersion
	EqualizeNodeWeights                 = shard.EqualizeNodeWeights
	FcapKeyPrefix                       = shard.FcapKeyPrefix
	FcapKeyPrefixSub                    = shard.FcapKeyPrefixSub
	FetchOpsSlotMapHTTP                 = shard.FetchOpsSlotMapHTTP
	FilterCampaignIDsBySlot             = shard.FilterCampaignIDsBySlot
	HomeSlotForCampaign                 = shard.HomeSlotForCampaign
	LoadActiveSlotMap                   = shard.LoadActiveSlotMap
	LoadOpsSlotMapFromPool              = shard.LoadOpsSlotMapFromPool
	MarshalEpochPayload                 = shard.MarshalEpochPayload
	MigrationFenceKeyPrefix             = shard.MigrationFenceKeyPrefix
	MigrationFenceRedisKey              = shard.MigrationFenceRedisKey
	NewCampaignRoutingRepo              = shard.NewCampaignRoutingRepo
	NewJumpHashSharder                  = shard.NewJumpHashSharder
	NewSlotMapRepo                      = shard.NewSlotMapRepo
	NewSlotMigrationRepo                = shard.NewSlotMigrationRepo
	NewStaticSlotSharder                = shard.NewStaticSlotSharder
	NodeWeightsDrainFrozen              = shard.NodeWeightsDrainFrozen
	NodeWeightsToJSON                   = shard.NodeWeightsToJSON
	PlacementBlacklistKey               = shard.PlacementBlacklistKey
	ProvenanceFromUDPCode               = shard.ProvenanceFromUDPCode
	ProvenanceToUDPCode                 = shard.ProvenanceToUDPCode
	PublishSlotMapReload                = shard.PublishSlotMapReload
	PublishSlotMigrationDeltaTestHelper = shard.PublishSlotMigrationDeltaTestHelper
	RedisClusterSlot                    = shard.RedisClusterSlot
	ReloadStaticSlotMapIfChanged        = shard.ReloadStaticSlotMapIfChanged
	RewarmCampaignBudgetKeys            = shard.RewarmCampaignBudgetKeys
	SetBudgetFrozen                     = shard.SetBudgetFrozen
	ShardFromSlotTable                  = shard.ShardFromSlotTable
	SlotMapShardTable                   = shard.SlotMapShardTable
	SlotMapsEqual                       = shard.SlotMapsEqual
	SlotMigrationDeltaCursorKey         = shard.SlotMigrationDeltaCursorKey
	SlotMigrationDeltaStreamKey         = shard.SlotMigrationDeltaStreamKey
	SlotMigrationDualWriteFlagKey       = shard.SlotMigrationDualWriteFlagKey
	SlotMigrationReplicationLag         = shard.SlotMigrationReplicationLag
	TableFromRows                       = shard.TableFromRows
	ToUUID                              = shard.ToUUID
)

const (
	SlotCount           = shard.SlotCount
	SlotMask            = shard.SlotMask
	UDPMaxControlShards = shard.UDPMaxControlShards
)

var (
	ErrSlotMapIncomplete      = shard.ErrSlotMapIncomplete
	ErrSlotMapVersionNotFound = shard.ErrSlotMapVersionNotFound
	ErrSlotMapInvalidSlot     = shard.ErrSlotMapInvalidSlot
	ErrSlotMapInvalidShard    = shard.ErrSlotMapInvalidShard
	ErrSlotMapAlreadyActive   = shard.ErrSlotMapAlreadyActive
)

var (
	UDPApplyCanaryFloor    = shard.UDPApplyCanaryFloor
	UDPDecodeHeader        = shard.UDPDecodeHeader
	UDPDecodeNodeWeights   = shard.UDPDecodeNodeWeights
	UDPDecodeShardLimits   = shard.UDPDecodeShardLimits
	UDPEncodeConfigRequest = shard.UDPEncodeConfigRequest
	UDPEncodeHeader        = shard.UDPEncodeHeader
	UDPEncodeNodeWeights   = shard.UDPEncodeNodeWeights
	UDPEncodeShardLimits   = shard.UDPEncodeShardLimits
	UDPFlagNodeWeights     = shard.UDPFlagNodeWeights
	UDPFlagSnapshot        = shard.UDPFlagSnapshot
	UDPHeaderSize          = shard.UDPHeaderSize
	UDPLimitsTightening    = shard.UDPLimitsTightening
	UDPMagic               = shard.UDPMagic
	UDPMaxNodeIDLen        = shard.UDPMaxNodeIDLen
	UDPMaxNodeWeights      = shard.UDPMaxNodeWeights
	UDPMsgConfigRequest    = shard.UDPMsgConfigRequest
	UDPMsgConfigSnapshot   = shard.UDPMsgConfigSnapshot
	UDPMsgMigrationBarrier = shard.UDPMsgMigrationBarrier
	UDPMsgQuotaEpoch       = shard.UDPMsgQuotaEpoch
	UDPNodeWeightStaleLag  = shard.UDPNodeWeightStaleLag
	UDPProtocolVersion     = shard.UDPProtocolVersion
	UDPProtocolVersion2    = shard.UDPProtocolVersion2
	UDPProtocolVersion3    = shard.UDPProtocolVersion3
	UDPShardPayloadLen     = shard.UDPShardPayloadLen
)
