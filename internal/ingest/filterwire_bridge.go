package ingest

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
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
	SegmentConversionHandler = fw.SegmentConversionHandler
)

var (
	NewFilterEngine             = fw.NewFilterEngine
	NewUnifiedFilter            = fw.NewUnifiedFilter
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
)

var filterEngineFailures = fw.FilterEngineFailures

type fraudAccumulator = fw.FraudAccumulator

func recordFraudMetrics(acc *fraudAccumulator, tier FraudTier, layer FraudLayer) {
	fw.RecordFraudMetricsLocal(acc, tier, layer)
}

func pingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
	return fw.PingConnectedRedisShards(ctx, redisShards)
}

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
