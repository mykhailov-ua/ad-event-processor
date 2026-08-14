package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

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
	rdb              redis.UniversalClient
	redisLoadTimeout time.Duration
	cache            atomic.Value
}

func NewBrandCreativeStore(rdb redis.UniversalClient, loadTimeoutMs int) *BrandCreativeStore {
	s := &BrandCreativeStore{rdb: rdb}
	if ms := FilterRedisReadTimeoutMs(loadTimeoutMs); ms > 0 {
		s.redisLoadTimeout = time.Duration(ms) * time.Millisecond
	}
	s.cache.Store(&brandCreativeMapSnapshot{byBrand: make(map[uuid.UUID][]brandCreativeEntry)})
	return s
}

func (s *BrandCreativeStore) LoadFromRedis(ctx context.Context, brandID uuid.UUID) {
	if s.rdb == nil {
		return
	}
	raw, err := s.rdb.Get(ctx, "brand:creatives:"+brandID.String()).Bytes()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			brandCreativeLoadTimeout.Inc()
		}
		return
	}
	var entries []brandCreativeEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		brandCreativeReplicaParseErrors.Inc()
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

func (s *BrandCreativeStore) SelectLandingURL(brandID uuid.UUID, userID string, evt *domain.Event) string {
	e, ok := s.selectCreative(brandID, userID, evt)
	if !ok {
		return ""
	}
	return e.URL
}

func (s *BrandCreativeStore) SelectLandingURLBytes(brandID uuid.UUID, userID string, evt *domain.Event) []byte {
	e, ok := s.selectCreative(brandID, userID, evt)
	if !ok {
		return nil
	}
	if len(e.urlBytes) > 0 {
		return e.urlBytes
	}
	if e.URL == "" {
		return nil
	}
	return UnsafeBytes(e.URL)
}

func (s *BrandCreativeStore) selectCreative(brandID uuid.UUID, userID string, evt *domain.Event) (brandCreativeEntry, bool) {
	entries := s.brandCreativeSnapshot().byBrand[brandID]
	if len(entries) == 0 && s.rdb != nil {
		s.loadFromRedisBounded(evt, brandID)
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
	_, _ = h.Write([]byte(brandID.String()))
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

func (s *BrandCreativeStore) loadFromRedisBounded(evt *domain.Event, brandID uuid.UUID) {
	if filterDeadlineExceededEvt(evt, nil) {
		brandCreativeLoadTimeout.Inc()
		return
	}
	rem, hasRem := filterDeadlineRemainingEvt(evt, nil)
	if hasRem && rem <= 0 {
		brandCreativeLoadTimeout.Inc()
		return
	}

	var ctx context.Context
	var cancel context.CancelFunc
	switch {
	case hasRem:
		ctx, cancel = context.WithTimeout(context.Background(), rem)
	case s.redisLoadTimeout > 0:
		ctx, cancel = context.WithTimeout(context.Background(), s.redisLoadTimeout)
	default:
		ctx = context.Background()
	}
	if cancel != nil {
		defer cancel()
	}
	s.LoadFromRedis(ctx, brandID)
}

type ScheduleFilter struct {
	registry domain.CampaignRegistry
}

func NewScheduleFilter(registry domain.CampaignRegistry) *ScheduleFilter {
	return &ScheduleFilter{registry: registry}
}

func (f *ScheduleFilter) Check(ctx context.Context, evt *domain.Event) error {
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok {
		if reg, ok := f.registry.(*Registry); ok && reg.IsStaleMode() {
			return ErrRegistryStale
		}
		return ErrCampaignNotFound
	}
	now := time.Now()
	if camp.StartAt != nil && now.Before(*camp.StartAt) {
		return ErrScheduleBlocked
	}
	if camp.EndAt != nil && !now.Before(*camp.EndAt) {
		return ErrScheduleBlocked
	}
	if len(camp.DaypartHours) > 0 {
		if camp.Location == nil {
			return ErrScheduleBlocked
		}
		hour := int16(now.In(camp.Location).Hour())
		if _, allowed := camp.DaypartHours[hour]; !allowed {
			return ErrScheduleBlocked
		}
	}
	return nil
}

func DaypartSliceToSet(hours []int16) map[int16]struct{} {
	if len(hours) == 0 {
		return nil
	}
	m := make(map[int16]struct{}, len(hours))
	for _, h := range hours {
		m[h] = struct{}{}
	}
	return m
}
