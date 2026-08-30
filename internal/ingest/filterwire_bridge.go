package ingest

import (
	"context"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	filterunified "ad-event-processor/internal/filter/unified"
	fw "ad-event-processor/internal/ingest/filterwire"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRateLimitExceeded      = fw.ErrRateLimitExceeded
	ErrDuplicateEvent         = fw.ErrDuplicateEvent
	ErrBudgetExhausted        = fw.ErrBudgetExhausted
	ErrCampaignNotFound       = fw.ErrCampaignNotFound
	ErrPacingExhausted        = fw.ErrPacingExhausted
	ErrFreqLimitExceeded      = fw.ErrFreqLimitExceeded
	ErrGeoBlocked             = fw.ErrGeoBlocked
	ErrScheduleBlocked        = fw.ErrScheduleBlocked
	ErrFraudDetected          = fw.ErrFraudDetected
	ErrEmergencyBreakerActive = fw.ErrEmergencyBreakerActive
	ErrBidFloorNotMet         = fw.ErrBidFloorNotMet
	ErrMigrationFenced        = fw.ErrMigrationFenced
	ErrLicenseExpired         = fw.ErrLicenseExpired
	ErrDailyQuotaExceeded     = fw.ErrDailyQuotaExceeded
	ErrRegistryStale          = fw.ErrRegistryStale
	ErrShardUnavailable       = fw.ErrShardUnavailable
	ErrInfraNetwork           = fw.ErrInfraNetwork
	ErrFilterTimeout          = fw.ErrFilterTimeout
	ErrPlacementBlocked       = fw.ErrPlacementBlocked
)

type (
	EventFilter              = fw.EventFilter
	FilterEngine             = fw.FilterEngine
	UnifiedFilter            = fw.UnifiedFilter
	FraudFilter              = fw.FraudFilter
	GeoFilter                = fw.GeoFilter
	BudgetFilter             = fw.BudgetFilter
	DuplicateEventFilter     = fw.DuplicateEventFilter
	EmergencyBreakerFilter   = fw.EmergencyBreakerFilter
	PlacementBlacklistFilter = fw.PlacementBlacklistFilter
	EntitlementsFilter       = fw.EntitlementsFilter
	FraudBlacklistFilter     = fw.FraudBlacklistFilter
	LicenseRPSFilter         = fw.LicenseRPSFilter
	ScheduleFilter           = fw.ScheduleFilter
	SegmentFilter            = fw.SegmentFilter
	DeviceFilter             = fw.DeviceFilter
	BehaviorTelemetryFilter  = fw.BehaviorTelemetryFilter
	JSONSerializationFilter  = fw.JSONSerializationFilter
	L7WireFilter             = fw.L7WireFilter
	TCPMSSFilter             = fw.TCPMSSFilter
	ResidentialProxyFilter   = fw.ResidentialProxyFilter
	LocalQuantaDeps          = fw.LocalQuantaDeps
	LocalTTCCache            = fw.LocalTTCCache
	RoughPacingGate          = fw.RoughPacingGate
	RedisBudgetManager       = fw.RedisBudgetManager
	IPRateLimiter            = fw.IPRateLimiter
	RedisStreamTrimmer       = fw.RedisStreamTrimmer
	RedisStreamTrimmerConfig = fw.RedisStreamTrimmerConfig
	DBHealthChecker          = fw.DBHealthChecker
	StringVal                = fw.StringVal
	BufWrapper               = fw.BufWrapper
	FraudTier                = fw.FraudTier
	FraudAccumulator         = fw.FraudAccumulator
	FraudLayer               = fw.FraudLayer
	ASNLookup                = fw.ASNLookup
	BrandCreativeStore       = fw.BrandCreativeStore
	BrandCreativeFixture     = fw.BrandCreativeFixture
	SegmentConversionHandler = fw.SegmentConversionHandler
)

var (
	NewFilterEngine             = fw.NewFilterEngine
	NewUnifiedFilter            = fw.NewUnifiedFilter
	NewDebitShardTestFilter     = fw.NewDebitShardTestFilter
	NewFraudFilter              = fw.NewFraudFilter
	NewGeoFilter                = fw.NewGeoFilter
	NewBudgetFilter             = fw.NewBudgetFilter
	NewDuplicateEventFilter     = fw.NewDuplicateEventFilter
	NewEmergencyBreakerFilter   = fw.NewEmergencyBreakerFilter
	NewPlacementBlacklistFilter = fw.NewPlacementBlacklistFilter
	NewEntitlementsFilter       = fw.NewEntitlementsFilter
	NewFraudBlacklistFilter     = fw.NewFraudBlacklistFilter
	NewLicenseRPSFilter         = fw.NewLicenseRPSFilter
	NewScheduleFilter           = fw.NewScheduleFilter
	NewSegmentFilter            = fw.NewSegmentFilter
	NewDeviceFilter             = fw.NewDeviceFilter
	NewBehaviorTelemetryFilter  = fw.NewBehaviorTelemetryFilter
	NewJSONSerializationFilter  = fw.NewJSONSerializationFilter
	NewL7WireFilter             = fw.NewL7WireFilter
	NewTCPMSSFilter             = fw.NewTCPMSSFilter
	NewResidentialProxyFilter   = fw.NewResidentialProxyFilter
	NewLocalTTCCache            = fw.NewLocalTTCCache
	NewRoughPacingGate          = fw.NewRoughPacingGate
	NewRedisBudgetManager       = fw.NewRedisBudgetManager
	NewIPRateLimiter            = fw.NewIPRateLimiter
	NewRedisStreamTrimmer       = fw.NewRedisStreamTrimmer
	InitUnifiedFilterLua        = fw.InitUnifiedFilterLua
	MapFraudTier                = fw.MapFraudTier
	AddFraudSignal              = fw.AddFraudSignal
	RecordFraudMetrics          = fw.RecordFraudMetrics
	NewBrandCreativeStore       = fw.NewBrandCreativeStore
	NewSegmentConversionHandler = fw.NewSegmentConversionHandler
	UnsafeString                = fw.UnsafeString
)

const (
	conversionEventType = fw.ConversionEventType
	connTimingRTTBit    = fw.ConnTimingRTTBit
	connTimingTTFBBit   = fw.ConnTimingTTFBBit
	hexChars            = fw.HexChars
	FraudTierPass       = fw.FraudTierPass
	FraudTierSuspect    = fw.FraudTierSuspect
	FraudTierIVT        = fw.FraudTierIVT
	FraudTierBlock      = fw.FraudTierBlock
	FraudLayerNone      = filter.FraudLayerNone
	FraudLayerL2Shadow  = filter.FraudLayerL2Shadow
	FraudLayerL1Reject  = filter.FraudLayerL1Reject
)

var filterEngineFailures = fw.FilterEngineFailures

type fraudAccumulator = fw.FraudAccumulator

func recordFraudMetrics(acc *fraudAccumulator, tier FraudTier, layer FraudLayer) {
	fw.RecordFraudMetricsLocal(acc, tier, layer)
}

func pingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
	return fw.PingConnectedRedisShards(ctx, redisShards)
}

func setFilterDeadlineOnEvent(evt *domain.Event, timeout time.Duration) {
	filter.SetFilterDeadlineOnEvent(evt, timeout)
}

func applyFraudAccumulatorForCampaign(evt *domain.Event, acc *fraudAccumulator, camp *domain.Campaign) FraudTier {
	return filter.ApplyFraudAccumulatorForCampaign(evt, acc, camp)
}

func decideFraudLayer(acc *fraudAccumulator, tier FraudTier) FraudLayer {
	return filter.DecideFraudLayer(acc, tier)
}

func applyFraudLayerDecision(evt *domain.Event, acc *fraudAccumulator, camp *domain.Campaign, boost uint8) (FraudLayer, error) {
	return filter.ApplyFraudLayerDecision(evt, acc, camp, boost)
}

func fraudThresholdsFromCampaign(camp *domain.Campaign) (pass, suspect, ivt, block uint8) {
	return filter.FraudThresholdsFromCampaign(camp)
}

func eventHasFraudL3(evt *domain.Event) bool {
	return filter.EventHasFraudL3(evt)
}

func licenseRPSSoftCeil(maxRPS uint64) uint64 {
	return filter.LicenseRPSSoftCeil(maxRPS)
}

func licenseRPSBurstCap(maxRPS uint64) uint64 {
	return filter.LicenseRPSBurstCap(maxRPS)
}

func resetGlobalDeploymentRPSForTests() {
	filter.ResetGlobalDeploymentRPSForTests()
}

func setGlobalDeploymentRPSBurstForTests(init uint32, remain uint64) {
	filter.SetGlobalDeploymentRPSBurstForTests(init, remain)
}

func globalDeploymentRPSBurstRemainForTests() uint64 {
	return filter.GlobalDeploymentRPSBurstRemainForTests()
}

func newRedisShardObservability(numShards int, sampleMask uint64) filter.RedisShardObservability {
	return filter.NewRedisShardObservability(numShards, sampleMask)
}

func sampledCampaignBucket(campaignID uuid.UUID) int {
	return filter.SampledCampaignBucket(campaignID)
}

func sampledCampaignBucketLabel(bucket int) string {
	return filter.SampledCampaignBucketLabel(bucket)
}

func spendMicroFromAny(amount any) int64 {
	return filterunified.SpendMicroFromAny(amount)
}

func normalizeRejectCountry(country string) string {
	return filter.NormalizeRejectCountry(country)
}

func appendRejectSamplePayload(dst []byte, kind, placement, country string) []byte {
	return filter.AppendRejectSamplePayload(dst, kind, placement, country)
}

func recordFilterRejectCountrySample(kind filter.FilterRejectKind, evt *domain.Event, seq *atomic.Uint64, sampleMask uint64) {
	filter.RecordFilterRejectCountrySample(kind, evt, seq, sampleMask)
}

func parseBlacklistUpdatePayload(payload string) (ip, reason string, ok bool) {
	return filter.ParseBlacklistUpdatePayload(payload)
}

func newFraudAccumulatorForTest(score uint32, signals ...FraudReasonID) *fraudAccumulator {
	ids := make([]filter.FraudReasonID, len(signals))
	for i, s := range signals {
		ids[i] = filter.FraudReasonID(s)
	}
	return filter.NewFraudAccumulatorForTest(score, ids...)
}

func filterDeadlineExceeded(ctx context.Context) bool {
	return filter.FilterDeadlineExceeded(ctx)
}

func filterDeadlineExceededEvt(evt *domain.Event, ctx context.Context) bool {
	return filter.FilterDeadlineExceededEvt(evt, ctx)
}

func filterDeadlineRemainingEvt(evt *domain.Event, ctx context.Context) (time.Duration, bool) {
	return filter.FilterDeadlineRemainingEvt(evt, ctx)
}

func filterDeadlineMonoFromContext(ctx context.Context) (int64, bool) {
	return filter.FilterDeadlineMonoFromContext(ctx)
}

var filterGeoLookupErrors = filter.FilterGeoLookupErrors

var filterFraudStreamWriteErrors = filter.FilterFraudStreamWriteErrors

func attachFilterDeadline(ctx context.Context, timeout time.Duration) context.Context {
	return filter.AttachFilterDeadline(ctx, timeout)
}

const fraudBlacklistKey = filter.FraudBlacklistKey

const (
	placementCacheShards             = filter.PlacementCacheShards
	placementCacheMaxEntriesPerShard = filter.PlacementCacheMaxEntriesPerShard
)

func attachFraudAccumulator(evt *domain.Event) *fraudAccumulator {
	return fw.AttachFraudAccumulator(evt)
}

func releaseFraudAccumulator(evt *domain.Event, acc *fraudAccumulator) {
	fw.ReleaseFraudAccumulator(evt, acc)
}

func segmentUserHash(hasher *piihash.Hasher, evt *domain.Event) ([16]byte, bool) {
	return fw.SegmentUserHash(hasher, evt)
}

func addSegmentMember(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte, ttl time.Duration) error {
	return fw.AddSegmentMember(ctx, redisShards, segmentID, userHash, ttl)
}

func segmentMemberExists(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte) (bool, error) {
	return fw.SegmentMemberExists(ctx, redisShards, segmentID, userHash)
}

func pickSegmentShard(redisShards []redis.UniversalClient, segmentID uuid.UUID) redis.UniversalClient {
	return fw.PickSegmentShard(redisShards, segmentID)
}

func firstConnectedRedisShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	return filterunified.FirstConnectedRedisShard(redisShards)
}

func tcpMSSWireValue(mss uint16) uint16 {
	return filterunified.TCPMSSWireValue(mss)
}

func addFraudSignal(evt *domain.Event, id FraudReasonID) {
	fw.AddFraudSignalID(evt, filter.FraudReasonID(id))
}

func parseBidMicro(payload []byte) int64 { return fw.ParseBidMicro(payload) }

func appendDate(dst []byte, t time.Time) []byte { return fw.AppendDate(dst, t) }

func appendUUID(dst []byte, u uuid.UUID) []byte {
	return filter.AppendUUID(dst, u)
}

func unsafeString(b []byte) string {
	return filter.UnsafeString(b)
}

type bufWrapper = BufWrapper

var bufPool = fw.BufPool
