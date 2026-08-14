package ingestion

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type errGeoProvider struct{}

var errGeoLookupFailed = errors.New("geo lookup failed")

func (errGeoProvider) GetCountry(ip string) (string, error) {
	return "", errGeoLookupFailed
}
func (errGeoProvider) IsAnonymous(ip string) (bool, error) { return false, nil }
func (errGeoProvider) Close() error                        { return nil }

func benchGeoFilterWithCountries(b *testing.B, geo GeoProvider) {
	campID := uuid.New()
	cachedMockCamp.Store(&domain.Campaign{
		ID:              campID,
		TargetCountries: map[string]struct{}{"US": {}},
	})
	b.Cleanup(func() { cachedMockCamp.Store(nil) })

	f := NewGeoFilter(geo, &mockRegistry{})
	evt := &domain.Event{
		IP:         "8.8.8.8",
		CampaignID: campID,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

// BenchmarkGeoFilter_lookupError measures GeoFilter with a failing geo provider.
// Harness: geo_mock_provider (errGeoProvider). Not production MaxMind perf.
// Production geo: BenchmarkGeoFilter_MaxMindCountry when deploy/geoip/GeoLite2-Country.mmdb present, or load test.
func BenchmarkGeoFilter_lookupError(b *testing.B) {
	benchGeoFilterWithCountries(b, errGeoProvider{})
}

// BenchmarkGeoFilter_lookupOK measures GeoFilter with MockGeoProvider.
// Harness: geo_mock_provider. Not production MaxMind perf.
// Production geo: BenchmarkGeoFilter_MaxMindCountry when .mmdb present, or load test.
func BenchmarkGeoFilter_lookupOK(b *testing.B) {
	benchGeoFilterWithCountries(b, &MockGeoProvider{})
}

// BenchmarkGeoFilter_MaxMindCountry measures GeoFilter with GeoLite2-Country.mmdb fixture.
// Harness: maxmind_mmdb_fixture. Skips when deploy/geoip/GeoLite2-Country.mmdb absent.
func BenchmarkGeoFilter_MaxMindCountry(b *testing.B) {
	const path = "deploy/geoip/GeoLite2-Country.mmdb"
	if _, err := os.Stat(path); err != nil {
		b.Skip("GeoLite2-Country.mmdb not present at " + path)
	}
	geo, err := NewMaxMindProvider(path)
	if err != nil {
		b.Fatalf("open mmdb: %v", err)
	}
	b.Cleanup(func() { _ = geo.Close() })
	benchGeoFilterWithCountries(b, geo)
}

func BenchmarkFraudFilter_DC(b *testing.B) {
	geo := &MockGeoProvider{}
	f := NewFraudFilter(geo)
	evt := &domain.Event{
		IP: "1.1.1.66",
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

// BenchmarkGeoFilter measures GeoFilter with MockGeoProvider and mockRegistry.
// Harness: geo_mock_provider. Not tracker p99 SLA evidence.
func BenchmarkGeoFilter(b *testing.B) {
	geo := &MockGeoProvider{}
	registry := &mockRegistry{}
	f := NewGeoFilter(geo, registry)
	evt := &domain.Event{
		IP:         "1.1.1.1",
		CampaignID: uuid.New(),
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkIPRateLimiter_Check(b *testing.B) {
	rdb := &mockRedisClient{}
	l := NewIPRateLimiter(rdb, 100, 10*time.Minute)
	evt := &domain.Event{
		IP: "192.168.1.1",
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Check(ctx, evt)
	}
}

func BenchmarkDuplicateEventFilter_Check(b *testing.B) {
	rdb := &mockRedisClient{}
	f := NewDuplicateEventFilter(rdb, 1*time.Hour)
	evt := &domain.Event{
		Type:    "click",
		ClickID: "click123",
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkKeyFormatting_impTSKey(b *testing.B) {
	evt := &domain.Event{
		UserID:     "user123",
		CampaignID: uuid.New(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := bufPool.Get().(*bufWrapper)
		w.buf = w.buf[:0]
		w.buf = append(w.buf, "imp_ts:"...)
		w.buf = append(w.buf, evt.UserID...)
		w.buf = append(w.buf, ':')
		w.buf = appendUUID(w.buf, evt.CampaignID)
		key := unsafeString(w.buf)
		_ = key
		bufPool.Put(w)
	}
}

func BenchmarkKeyFormatting_IPRateLimiter(b *testing.B) {
	evt := &domain.Event{
		IP: "192.168.1.1",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := bufPool.Get().(*bufWrapper)
		w.buf = w.buf[:0]
		w.buf = append(w.buf, "ratelimit:ip:"...)
		w.buf = append(w.buf, evt.IP...)
		key := unsafeString(w.buf)
		_ = key
		bufPool.Put(w)
	}
}

func BenchmarkKeyFormatting_DuplicateEventFilter(b *testing.B) {
	evt := &domain.Event{
		Type:    "click",
		ClickID: "click123",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := bufPool.Get().(*bufWrapper)
		w.buf = w.buf[:0]
		w.buf = append(w.buf, "dup:"...)
		w.buf = append(w.buf, evt.Type...)
		w.buf = append(w.buf, ':')
		w.buf = append(w.buf, evt.ClickID...)
		key := unsafeString(w.buf)
		_ = key
		bufPool.Put(w)
	}
}

// BenchmarkUnifiedFilter_Check measures UnifiedFilter.Check Go-layer overhead only.
// Harness: unified_filter_mock_redis — mockRedisClient.EvalSha does not execute Lua.
// For unified-filter Lua latency use BenchmarkLuaScript_Happy (testcontainers Redis).
func BenchmarkUnifiedFilter_Check(b *testing.B) {
	rdb := &mockRedisClient{}
	sharder := NewJumpHashSharder(1)
	registry := &mockRegistry{}

	f := NewUnifiedFilter(
		[]redis.UniversalClient{rdb},
		sharder,
		registry,
		nil,
		100,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		10000,
	)

	evt := &domain.Event{
		Type:       "click",
		IP:         "1.1.1.1",
		UserID:     "user123",
		CampaignID: uuid.New(),
		ClickID:    "click123",
	}
	setFilterDeadlineOnEvent(evt, time.Second)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

// BenchmarkRedisBudgetManager_CheckAndSpend measures CheckAndSpend Go-layer overhead only.
// Harness: redis_budget_mock — mockRedisClient.EvalSha does not execute Lua.
// For unified-filter Lua latency use BenchmarkLuaScript_Happy (testcontainers Redis).
func BenchmarkRedisBudgetManager_CheckAndSpend(b *testing.B) {
	rdb := &mockRedisClient{}
	bm := NewRedisBudgetManager(rdb, nil, time.Hour)

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	clickID := "click123"
	amount := int64(100_000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bm.CheckAndSpend(ctx, customerID, campaignID, clickID, amount)
	}
}
