package filterwire

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/filter"
	filterunified "ad-event-processor/internal/filter/unified"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRateLimitExceeded      = filter.ErrRateLimitExceeded
	ErrDuplicateEvent         = filter.ErrDuplicateEvent
	ErrBudgetExhausted        = filter.ErrBudgetExhausted
	ErrCampaignNotFound       = filter.ErrCampaignNotFound
	ErrPacingExhausted        = filter.ErrPacingExhausted
	ErrFreqLimitExceeded      = filter.ErrFreqLimitExceeded
	ErrGeoBlocked             = filter.ErrGeoBlocked
	ErrScheduleBlocked        = filter.ErrScheduleBlocked
	ErrFraudDetected          = filter.ErrFraudDetected
	ErrEmergencyBreakerActive = filter.ErrEmergencyBreakerActive
	ErrBidFloorNotMet         = filter.ErrBidFloorNotMet
	ErrMigrationFenced        = filter.ErrMigrationFenced
	ErrLicenseExpired         = filter.ErrLicenseExpired
	ErrDailyQuotaExceeded     = filter.ErrDailyQuotaExceeded
	ErrRegistryStale          = filter.ErrRegistryStale
	ErrShardUnavailable       = filter.ErrShardUnavailable
	ErrInfraNetwork           = filter.ErrInfraNetwork
	ErrFilterTimeout          = filter.ErrFilterTimeout
	ErrPlacementBlocked       = filter.ErrPlacementBlocked
)

type (
	EventFilter              = filter.EventFilter
	FilterEngine             = filter.FilterEngine
	UnifiedFilter            = filterunified.UnifiedFilter
	FraudFilter              = filter.FraudFilter
	GeoFilter                = filter.GeoFilter
	BudgetFilter             = filter.BudgetFilter
	DuplicateEventFilter     = filter.DuplicateEventFilter
	EmergencyBreakerFilter   = filter.EmergencyBreakerFilter
	PlacementBlacklistFilter = filter.PlacementBlacklistFilter
	EntitlementsFilter       = filter.EntitlementsFilter
	FraudBlacklistFilter     = filter.FraudBlacklistFilter
	LicenseRPSFilter         = filter.LicenseRPSFilter
	ScheduleFilter           = filter.ScheduleFilter
	SegmentFilter            = filter.SegmentFilter
	DeviceFilter             = filter.DeviceFilter
	BehaviorTelemetryFilter  = filter.BehaviorTelemetryFilter
	JSONSerializationFilter  = filterunified.JSONSerializationFilter
	L7WireFilter             = filterunified.L7WireFilter
	TCPMSSFilter             = filterunified.TCPMSSFilter
	ResidentialProxyFilter   = filterunified.ResidentialProxyFilter
	LocalQuantaDeps          = filterunified.LocalQuantaDeps
	LocalTTCCache            = filterunified.LocalTTCCache
	RoughPacingGate          = filterunified.RoughPacingGate
	RedisBudgetManager       = filterunified.RedisBudgetManager
	IPRateLimiter            = filterunified.IPRateLimiter
	RedisStreamTrimmer       = filterunified.RedisStreamTrimmer
	RedisStreamTrimmerConfig = filterunified.RedisStreamTrimmerConfig
	DBHealthChecker          = filter.DBHealthChecker
	StringVal                = filter.StringVal
	BufWrapper               = filter.BufWrapper
	FraudTier                = filter.FraudTier
	FraudAccumulator         = filter.FraudAccumulator
	FraudLayer               = filter.FraudLayer
	ASNLookup                = filter.ASNLookup
)

var (
	NewFilterEngine             = filter.NewFilterEngine
	NewUnifiedFilter            = filterunified.NewUnifiedFilter
	NewDebitShardTestFilter     = filterunified.NewDebitShardTestFilter
	NewFraudFilter              = filter.NewFraudFilter
	NewGeoFilter                = filter.NewGeoFilter
	NewBudgetFilter             = filter.NewBudgetFilter
	NewDuplicateEventFilter     = filter.NewDuplicateEventFilter
	NewEmergencyBreakerFilter   = filter.NewEmergencyBreakerFilter
	NewPlacementBlacklistFilter = filter.NewPlacementBlacklistFilter
	NewEntitlementsFilter       = filter.NewEntitlementsFilter
	NewFraudBlacklistFilter     = filter.NewFraudBlacklistFilter
	NewLicenseRPSFilter         = filter.NewLicenseRPSFilter
	NewScheduleFilter           = filter.NewScheduleFilter
	NewSegmentFilter            = filter.NewSegmentFilter
	NewDeviceFilter             = filter.NewDeviceFilter
	NewBehaviorTelemetryFilter  = filter.NewBehaviorTelemetryFilter
	NewJSONSerializationFilter  = filterunified.NewJSONSerializationFilter
	NewL7WireFilter             = filterunified.NewL7WireFilter
	NewTCPMSSFilter             = filterunified.NewTCPMSSFilter
	NewResidentialProxyFilter   = filterunified.NewResidentialProxyFilter
	NewLocalTTCCache            = filterunified.NewLocalTTCCache
	NewRoughPacingGate          = filterunified.NewRoughPacingGate
	NewRedisBudgetManager       = filterunified.NewRedisBudgetManager
	NewIPRateLimiter            = filterunified.NewIPRateLimiter
	NewRedisStreamTrimmer       = filterunified.NewRedisStreamTrimmer
	InitUnifiedFilterLua        = filterunified.InitUnifiedFilterLua
	MapFraudTier                = filter.MapFraudTier
	AddFraudSignal              = filter.AddFraudSignal
	RecordFraudMetrics          = filter.RecordFraudMetrics
)

func appendUUID(dst []byte, u uuid.UUID) []byte {
	return filter.AppendUUID(dst, u)
}

func unsafeString(b []byte) string {
	return filter.UnsafeString(b)
}

type bufWrapper = BufWrapper

var BufPool = sync.Pool{
	New: func() any {
		return &BufWrapper{Buf: make([]byte, 0, 128)}
	},
}

type brandCreativeEntry struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Weight   int32  `json:"weight"`
	urlBytes []byte
}

func brandCreativeEntriesReady(entries []brandCreativeEntry) []brandCreativeEntry {
	for i := range entries {
		if entries[i].URL != "" && len(entries[i].urlBytes) == 0 {
			entries[i].urlBytes = []byte(entries[i].URL)
		}
	}
	return entries
}

type brandCreativeMapSnapshot struct {
	byBrand map[uuid.UUID][]brandCreativeEntry
}

func (s *BrandCreativeStore) brandCreativeSnapshot() *brandCreativeMapSnapshot {
	v, ok := s.cache.Load().(*brandCreativeMapSnapshot)
	if !ok || v == nil {
		return &brandCreativeMapSnapshot{}
	}
	return v
}

type BrandCreativeStore struct {
	redisClient      redis.UniversalClient
	redisLoadTimeout time.Duration
	cache            atomic.Value
}

func NewBrandCreativeStore(redisClient redis.UniversalClient, loadTimeoutMs int) *BrandCreativeStore {
	s := &BrandCreativeStore{redisClient: redisClient}
	if ms := filter.FilterRedisReadTimeoutMs(loadTimeoutMs); ms > 0 {
		s.redisLoadTimeout = time.Duration(ms) * time.Millisecond
	}
	s.cache.Store(&brandCreativeMapSnapshot{byBrand: make(map[uuid.UUID][]brandCreativeEntry)})
	return s
}

type BrandCreativeFixture struct {
	ID     string
	URL    string
	Weight int32
}

func (s *BrandCreativeStore) SetFixturesForTest(byBrand map[uuid.UUID][]BrandCreativeFixture) {
	if s == nil {
		return
	}
	next := make(map[uuid.UUID][]brandCreativeEntry, len(byBrand))
	for brandID, fixtures := range byBrand {
		entries := make([]brandCreativeEntry, 0, len(fixtures))
		for _, f := range fixtures {
			entries = append(entries, brandCreativeEntry{
				ID:     f.ID,
				URL:    f.URL,
				Weight: f.Weight,
			})
		}
		next[brandID] = brandCreativeEntriesReady(entries)
	}
	s.cache.Store(&brandCreativeMapSnapshot{byBrand: next})
}

func (s *BrandCreativeStore) LoadFromRedis(ctx context.Context, brandID uuid.UUID) {
	if s.redisClient == nil {
		return
	}
	raw, err := s.redisClient.Get(ctx, "brand:creatives:"+brandID.String()).Bytes()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			filter.BrandCreativeLoadTimeout.Inc()
		}
		return
	}
	var entries []brandCreativeEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		filter.BrandCreativeReplicaParseErrors.Inc()
		slog.Warn("brand creative replica corrupt", "brand_id", brandID, "error", err)
		return
	}
	entries = brandCreativeEntriesReady(entries)
	current := s.brandCreativeSnapshot().byBrand
	next := make(map[uuid.UUID][]brandCreativeEntry, len(current)+1)
	for k, v := range current {
		next[k] = v
	}
	next[brandID] = entries
	s.cache.Store(&brandCreativeMapSnapshot{byBrand: next})
}

func (s *BrandCreativeStore) SelectLandingURL(ctx context.Context, brandID uuid.UUID, userID string, evt *domain.Event) string {
	e, ok := s.selectCreative(ctx, brandID, userID, evt)
	if !ok {
		return ""
	}
	return e.URL
}

func (s *BrandCreativeStore) SelectLandingURLBytes(ctx context.Context, brandID uuid.UUID, userID string, evt *domain.Event) []byte {
	e, ok := s.selectCreative(ctx, brandID, userID, evt)
	if !ok {
		return nil
	}
	if len(e.urlBytes) > 0 {
		return e.urlBytes
	}
	if e.URL == "" {
		return nil
	}
	return filter.UnsafeBytes(e.URL)
}

func (s *BrandCreativeStore) selectCreative(ctx context.Context, brandID uuid.UUID, userID string, evt *domain.Event) (brandCreativeEntry, bool) {
	entries := s.brandCreativeSnapshot().byBrand[brandID]
	if len(entries) == 0 && s.redisClient != nil {
		s.loadFromRedisBounded(ctx, evt, brandID)
		entries = s.brandCreativeSnapshot().byBrand[brandID]
	}
	if len(entries) == 0 {
		return brandCreativeEntry{}, false
	}
	if len(entries) == 1 {
		return entries[0], true
	}

	total := int32(0)
	for _, e := range entries {
		total += e.Weight
	}
	if total <= 0 {
		return entries[0], true
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	var brandScratch [36]byte
	_, _ = h.Write(appendUUID(brandScratch[:0], brandID))
	bucket := int32(h.Sum32() % uint32(total))

	var acc int32
	for _, e := range entries {
		acc += e.Weight
		if bucket < acc {
			return e, true
		}
	}
	return entries[len(entries)-1], true
}

func (s *BrandCreativeStore) loadFromRedisBounded(ctx context.Context, evt *domain.Event, brandID uuid.UUID) {
	if filter.FilterDeadlineExceededOnEvent(evt, ctx) {
		filter.BrandCreativeLoadTimeout.Inc()
		return
	}
	rem, hasRem := filter.FilterDeadlineRemainingOnEvent(evt, ctx)
	if hasRem && rem <= 0 {
		filter.BrandCreativeLoadTimeout.Inc()
		return
	}

	var loadCtx context.Context
	var cancel context.CancelFunc
	switch {
	case hasRem:
		loadCtx, cancel = context.WithTimeout(ctx, rem)
	case s.redisLoadTimeout > 0:
		loadCtx, cancel = context.WithTimeout(ctx, s.redisLoadTimeout)
	default:
		loadCtx = ctx
	}
	if cancel != nil {
		defer cancel()
	}
	s.LoadFromRedis(loadCtx, brandID)
}

type segmentCampaignLoader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type SegmentConversionHandler struct {
	repo        segmentCampaignLoader
	queries     db.Querier
	redisShards []redis.UniversalClient
	hasher      *piihash.Hasher
}

func NewSegmentConversionHandler(repo segmentCampaignLoader, queries db.Querier, redisShards []redis.UniversalClient, hasher *piihash.Hasher) *SegmentConversionHandler {
	return &SegmentConversionHandler{
		repo:        repo,
		queries:     queries,
		redisShards: redisShards,
		hasher:      hasher,
	}
}

func (h *SegmentConversionHandler) Handle(evt *domain.Event, _ string) {
	if h == nil || evt == nil || evt.Type != ConversionEventType || evt.UserID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	camp, err := h.repo.GetByID(ctx, evt.CampaignID)
	if err != nil || camp == nil || camp.RetargetSegmentID == uuid.Nil {
		return
	}
	ttlHours := camp.SegmentTTLHours
	if ttlHours <= 0 {
		return
	}
	userHash, ok := SegmentUserHash(h.hasher, evt)
	if !ok {
		return
	}
	ttl := time.Duration(ttlHours) * time.Hour
	if err := AddSegmentMember(ctx, h.redisShards, camp.RetargetSegmentID, userHash, ttl); err != nil {
		return
	}
	if h.queries == nil {
		return
	}
	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(ttl), Valid: true}
	_ = h.queries.UpsertSegmentMember(ctx, db.UpsertSegmentMemberParams{
		SegmentID: pgtype.UUID{Bytes: camp.RetargetSegmentID, Valid: true},
		UserHash:  userHash[:],
		ExpiresAt: expiresAt,
	})
}

func RecordFraudMetricsLocal(acc *FraudAccumulator, tier FraudTier, layer FraudLayer) {
	filter.RecordFraudMetrics(acc, tier, layer)
}

func PingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
	return filterunified.PingConnectedRedisShards(ctx, redisShards)
}

func AttachFraudAccumulator(evt *domain.Event) *FraudAccumulator {
	return filter.AttachFraudAccumulator(evt)
}

func ReleaseFraudAccumulator(evt *domain.Event, acc *FraudAccumulator) {
	filter.ReleaseFraudAccumulator(evt, acc)
}

const ConversionEventType = filter.ConversionEventType

func SegmentUserHash(hasher *piihash.Hasher, evt *domain.Event) ([16]byte, bool) {
	return filter.SegmentUserHash(hasher, evt)
}

func AddSegmentMember(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte, ttl time.Duration) error {
	return filter.AddSegmentMember(ctx, redisShards, segmentID, userHash, ttl)
}

func SegmentMemberExists(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte) (bool, error) {
	return filter.SegmentMemberExists(ctx, redisShards, segmentID, userHash)
}

func PickSegmentShard(redisShards []redis.UniversalClient, segmentID uuid.UUID) redis.UniversalClient {
	return filter.PickSegmentShard(redisShards, segmentID)
}

func AddFraudSignalID(evt *domain.Event, id filter.FraudReasonID) {
	filter.AddFraudSignal(evt, id)
}

const (
	ConnTimingRTTBit  = filter.ConnTimingRTTBit
	ConnTimingTTFBBit = filter.ConnTimingTTFBBit
	HexChars          = filter.HexChars
)

const (
	FraudTierPass    = filter.FraudTierPass
	FraudTierSuspect = filter.FraudTierSuspect
	FraudTierIVT     = filter.FraudTierIVT
	FraudTierBlock   = filter.FraudTierBlock
)

var UnsafeString = filter.UnsafeString

func ParseBidMicro(payload []byte) int64        { return filterunified.ParseBidMicro(payload) }
func AppendDate(dst []byte, t time.Time) []byte { return filter.AppendDate(dst, t) }

var FilterEngineFailures = filter.FilterEngineFailures
