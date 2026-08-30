package ingest

import (
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	compat "ad-event-processor/internal/ingest/compat"
	"ad-event-processor/internal/stream"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
)

type (
	filterRejectKind = filter.FilterRejectKind
	cidrBuilder      = filter.CIDRBuilder
	mockRegistry     = filter.MockRegistry
	cidrNode         = filter.CIDRNode
	slotTable        = domain.SlotTable
)

const cidrNoIndex = filter.CIDRNoIndex

var (
	cidrFeedNames                 = filter.CIDRFeedNames
	cachedMockCamp                = filter.CachedMockCamp()
	buildDCASNSnapshot            = filter.BuildDCASNSnapshot
	shouldBypassCGNATIPVelocity   = filter.ShouldBypassCGNATIPVelocity
	hexByte                       = filter.HexByte
	scanUAFamily                  = filter.ScanUAFamily
	normalizeCapturedTTL          = filter.NormalizeCapturedTTL
	campaignHashTag               = domain.CampaignHashTag
	budgetCampaignKey             = domain.BudgetCampaignKey
	customerSyncKey               = domain.CustomerSyncKey
	fcapKeyPrefix                 = domain.FcapKeyPrefix
	dailySpendKeyPrefix           = domain.DailySpendKeyPrefix
	classifyFilterErr             = filter.ClassifyFilterErr
	parseASNLine                  = filter.ParseASNLine
	enrichMockCampaign            = filter.EnrichMockCampaign
	lockStaticCampaign            = filter.LockStaticCampaign
	configureMockRegistryCampaign = filter.ConfigureMockRegistryCampaign
	resetStaticCampaignBaseline   = filter.ResetStaticCampaignBaseline
	budgetQuotaKey                = filter.BudgetQuotaKey
	buildSlotTable                = domain.BuildSlotTable
	cachedUnixMilliNow            = filter.CachedUnixMilliNow
	getCampaignFromEvent          = filter.GetCampaignFromEvent
	asnLookupFromGeo              = filter.AsnLookupFromGeo
	shouldSampleLuaMetrics        = filter.ShouldSampleLuaMetrics
	auditLogSampleMaskFromConfig  = stream.AuditLogSampleMaskFromConfig
	applyMobileBiometricSummary   = filter.ApplyMobileBiometricSummary
	parseTCPSigHeader             = filter.ParseTCPSigHeader
	uaMatchesInAppWebView         = filter.UAMatchesInAppWebView
	osFingerprintMismatch         = filter.OsFingerprintMismatch
	tcpSynSigMismatch             = filter.TcpSynSigMismatch
	hashTCPSynFields              = filter.HashTCPSynFields
	monotonicNano                 = filter.MonotonicNano
	monoElapsedSeconds            = filter.MonoElapsedSeconds
	matchUAAt                     = filter.MatchUAAt
	campaignSyncKey               = domain.CampaignSyncKey
	timezoneMismatchHours         = filter.TimezoneMismatchHours
	marshalCHSpoolPayload         = stream.MarshalCHSpoolPayload
	crc32Castagnoli               = filter.CRC32Castagnoli
	openRTBLicenseAllowed         = filter.OpenRTBLicenseAllowed
	parseIPv6To128                = stream.ParseIPv6To128
	appendCampaignHashTag         = filter.AppendCampaignHashTag
	loadTCPSynSigCorpusFromDir    = filter.LoadTCPSynSigCorpusFromDir
	cachedUnixMilliLoad           = filter.CachedUnixMilliNow
	cachedUnixMilliStore          = filter.CachedUnixMilliStore
	cachedUnixMilliAnyStore       = filter.CachedUnixMilliAnyStore
	cachedNowUTCSetFromUnixMilli  = filter.CachedNowUTCSetFromUnixMilli
	storeCachedNowUTC             = filter.StoreCachedNowUTC
	setClockRefreshPaused         = filter.SetClockRefreshPaused
	cgnatBypassForCampaign        = filter.CgnatBypassForCampaign
	histogramSampleMaskFromConfig = filter.HistogramSampleMaskFromConfig
	cachedUnixMilliAnyLoad        = filter.CachedUnixMilliAnyLoad
	observeHistogramSampled       = filter.ObserveHistogramSampled
)

const (
	uaFamilyWindows           = filter.UAFamilyWindows
	uaFamilyMac               = filter.UAFamilyMac
	uaFamilyLinux             = filter.UAFamilyLinux
	uaFamilyMobile            = filter.UAFamilyMobile
	fraudReasonCount          = filter.FraudReasonID(filter.FraudReasonCount)
	fraudSignalL1High         = filter.FraudSignalL1High
	fraudSignalL3             = filter.FraudSignalL3
	luaMetricsSampleMask      = filter.LuaMetricsSampleMask
	auditLogSampleMaskDefault = 127
	fraudSignalL2Weak         = filter.FraudSignalL2Weak
	defaultLatencyRingCap     = filter.DefaultLatencyRingCap
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

func writeAuditLog(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
) {
	compat.WriteAuditLog(l, seq, sampleMask, shardID, evt)
}

func enqueueFraudReject(writer *FraudStreamWriter, shard int, evt *domain.Event) {
	compat.EnqueueFraudReject(writer, shard, evt)
}
