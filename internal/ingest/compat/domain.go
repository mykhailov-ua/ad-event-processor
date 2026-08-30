package compat

import (
	"net/netip"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/budget"
	"ad-event-processor/internal/domain/shard"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/internal/stream"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	FraudReasonID    = filter.FraudReasonID
	filterRejectKind = filter.FilterRejectKind
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

const (
	fraudSignalL1High = filter.FraudSignalL1High
	fraudSignalL3     = filter.FraudSignalL3
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

func buildDCASNSnapshot(asns map[uint32]struct{}, gen uint64) *filter.DCASNSnapshot {
	return filter.BuildDCASNSnapshot(asns, gen)
}

var BuildDCASNSnapshot = filter.BuildDCASNSnapshot

func shouldBypassCGNATIPVelocity(
	globalBypass bool,
	camp *domain.Campaign,
	carrierTable *MobileCarrierASNTable,
	lookup filter.ASNLookup,
	ip string,
	signal string,
) bool {
	return filter.ShouldBypassCGNATIPVelocity(globalBypass, camp, carrierTable, lookup, ip, signal)
}

type cidrBuilder = filter.CIDRBuilder

const cidrNoIndex = filter.CIDRNoIndex

func HashResidentialProxyUser(s string) uint32 {
	return filter.HashResidentialProxyUser(s)
}

func HashResidentialProxyUA(s string) uint32 {
	return filter.HashResidentialProxyUA(s)
}

func RemainingBudgetMicro(camp *domain.Campaign) int64 {
	return filter.RemainingBudgetMicro(camp)
}

func matchUAAt(ua string, i, n int, needle string) bool {
	return filter.MatchUAAt(ua, i, n, needle)
}

const uaScanMax = filter.UAScanMax

func hexByte(n byte) byte {
	return filter.HexByte(n)
}

func scanUAFamily(ua string) uint8 {
	return filter.ScanUAFamily(ua)
}

var ScanUAFamily = filter.ScanUAFamily

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

func normalizeCapturedTTL(captured uint8) uint8 {
	return filter.NormalizeCapturedTTL(captured)
}

func campaignHashTag(id uuid.UUID) string {
	return domain.CampaignHashTag(id)
}

func budgetCampaignKey(id uuid.UUID) string {
	return domain.BudgetCampaignKey(id)
}

func campaignSyncKey(id uuid.UUID) string {
	return domain.CampaignSyncKey(id)
}

var CampaignSyncKey = domain.CampaignSyncKey

func customerSyncKey(campaignID, customerID uuid.UUID) string {
	return domain.CustomerSyncKey(campaignID, customerID)
}

func fcapKeyPrefix(campaignID uuid.UUID, brandFcapKey string) string {
	return domain.FcapKeyPrefix(campaignID, brandFcapKey)
}

func dailySpendKeyPrefix(campaignID uuid.UUID) string {
	return domain.DailySpendKeyPrefix(campaignID)
}

var RedisClusterSlot = domain.RedisClusterSlot

func timezoneMismatchHours(browserTZ, country string, now time.Time) (bool, int) {
	return filter.TimezoneMismatchHours(browserTZ, country, now)
}

var (
	ErrSegmentNotIncluded = filter.ErrSegmentNotIncluded
	ErrSegmentExcluded    = filter.ErrSegmentExcluded
)

type mockRegistry = filter.MockRegistry

var (
	ErrConsentDenied                = filter.ErrConsentDenied
	FraudReasonCodeDatacenterIP     = filter.FraudReasonCodeDatacenterIP
	FraudReasonCodeL3Blocklist      = filter.FraudReasonCodeL3Blocklist
	FraudReasonCodeIPv4Rotation     = filter.FraudReasonCodeIPv4Rotation
	FraudReasonCodeTCPSynOSMismatch = filter.FraudReasonCodeTCPSynOSMismatch
	ClassifyFilterErr               = filter.ClassifyFilterErr
	ParseASNLine                    = filter.ParseASNLine
)

func classifyFilterErr(err error) (filter.FilterRejectKind, bool) {
	return filter.ClassifyFilterErr(err)
}

func parseASNLine(line string) (uint32, bool) {
	return filter.ParseASNLine(line)
}

func enrichMockCampaign(cp *domain.Campaign) {
	filter.EnrichMockCampaign(cp)
}

func lockStaticCampaign(mut func(c *domain.Campaign)) {
	filter.LockStaticCampaign(mut)
}

func WithStaticCampaign(fn func(camp **domain.Campaign)) {
	filter.WithStaticCampaign(fn)
}

func configureMockRegistryCampaign(mut func(c *domain.Campaign)) {
	filter.ConfigureMockRegistryCampaign(mut)
}

func resetStaticCampaignBaseline() {
	filter.ResetStaticCampaignBaseline()
}

var cachedMockCamp = filter.CachedMockCamp()

func marshalCHSpoolPayload(dedupToken string, events []*domain.Event) ([]byte, error) {
	return stream.MarshalCHSpoolPayload(dedupToken, events)
}

func crc32Castagnoli(data *uuid.UUID) uint32 {
	return filter.CRC32Castagnoli(data)
}

var (
	CRC32Castagnoli       = filter.CRC32Castagnoli
	AppendCampaignHashTag = filter.AppendCampaignHashTag
	PlacementBlacklistKey = filter.PlacementBlacklistKey
)

func openRTBLicenseAllowed(reg domain.CampaignRegistry) bool {
	return filter.OpenRTBLicenseAllowed(reg)
}

func parseIPv6To128(ip string) (hi, lo uint64, ok bool) {
	return stream.ParseIPv6To128(ip)
}

func appendCampaignHashTag(dst []byte, id uuid.UUID) []byte {
	return filter.AppendCampaignHashTag(dst, id)
}

func budgetQuotaKey(id uuid.UUID) string {
	return filter.BudgetQuotaKey(id)
}

func WriteAuditLog(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
) {
	stream.WriteAuditLog(l, seq, sampleMask, shardID, evt)
}

func auditEventFromFields(ts int64, campaignID uuid.UUID, clickID, eventType string) *domain.Event {
	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()
	evt.ClickID = clickID
	evt.CampaignID = campaignID
	evt.Type = eventType
	if ts > 0 {
		evt.CreatedAt = time.Unix(ts, 0)
	}
	return evt
}

func EnqueueFraudReject(writer *stream.FraudStreamWriter, shard int, evt *domain.Event) {
	stream.EnqueueFraudReject(writer, shard, evt)
}

const (
	FraudReasonCodeMissingImpTS  = filter.FraudReasonCodeMissingImpTS
	FraudReasonCodeLowTTC        = filter.FraudReasonCodeLowTTC
	FraudReasonCodeOSFingerprint = filter.FraudReasonCodeOSFingerprint
)

func monotonicNano() int64 {
	return filter.MonotonicNano()
}

func loadTCPSynSigCorpusFromDir(dir string) *filter.TCPSynSigCorpusSnapshot {
	return filter.LoadTCPSynSigCorpusFromDir(dir)
}

func PublishTCPSynSigCorpus(snap *filter.TCPSynSigCorpusSnapshot) {
	filter.PublishTCPSynSigCorpus(snap)
}

func cachedUnixMilliLoad() int64 {
	return filter.CachedUnixMilliNow()
}

func cachedUnixMilliStore(ms int64) {
	filter.CachedUnixMilliStore(ms)
}

func cachedUnixMilliAnyStore(v any) {
	filter.CachedUnixMilliAnyStore(v)
}

func cachedNowUTCSetFromUnixMilli(ms int64) {
	filter.CachedNowUTCSetFromUnixMilli(ms)
}

func storeCachedNowUTC() {
	filter.StoreCachedNowUTC()
}

func setClockRefreshPaused(paused bool) {
	filter.SetClockRefreshPaused(paused)
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

type slotTable = domain.SlotTable

func buildSlotTable(numBuckets int) *slotTable {
	return domain.BuildSlotTable(numBuckets)
}

type IngressQuotaMap = stream.IngressQuotaMap

func BuildIngressQuotaMap(epoch int64, limits *UDPControlLimits, numWorkers int) *IngressQuotaMap {
	return stream.BuildIngressQuotaMap(epoch, limits, numWorkers)
}
