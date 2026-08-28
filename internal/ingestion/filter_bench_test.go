package ingestion

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type errGeoProvider struct{}

var errGeoLookupFailed = errors.New("geo lookup failed")

func (p errGeoProvider) GetCountry(ip string) (string, error) {
	return "", errGeoLookupFailed
}
func (p errGeoProvider) IsAnonymous(ip string) (bool, error) { return false, nil }
func (p errGeoProvider) Close() error                        { return nil }

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
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkGeoFilter_lookupError(b *testing.B) {
	benchGeoFilterWithCountries(b, errGeoProvider{})
}

func BenchmarkGeoFilter_lookupOK(b *testing.B) {
	benchGeoFilterWithCountries(b, &MockGeoProvider{})
}

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
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkGeoFilter(b *testing.B) {
	geo := &MockGeoProvider{}
	registry := &mockRegistry{}
	f := NewGeoFilter(geo, registry)
	evt := &domain.Event{
		IP:         "1.1.1.1",
		CampaignID: uuid.New(),
	}
	ctx := context.Background()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkIPRateLimiter_Check(b *testing.B) {
	redisClient := &mockRedisClient{}
	l := NewIPRateLimiter(redisClient, 100, 10*time.Minute)
	evt := &domain.Event{
		IP: "192.168.1.1",
	}
	ctx := context.Background()
	for b.Loop() {
		_ = l.Check(ctx, evt)
	}
}

func BenchmarkDuplicateEventFilter_Check(b *testing.B) {
	redisClient := &mockRedisClient{}
	f := NewDuplicateEventFilter(redisClient, 1*time.Hour)
	evt := &domain.Event{
		Type:    "click",
		ClickID: "click123",
	}
	ctx := context.Background()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkKeyFormatting_impTSKey(b *testing.B) {
	evt := &domain.Event{
		UserID:     "user123",
		CampaignID: uuid.New(),
	}
	for b.Loop() {
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
	for b.Loop() {
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
	for b.Loop() {
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

func BenchmarkUnifiedFilter_Check_mock(b *testing.B) {
	redisClient := &mockRedisClient{}
	sharder := NewJumpHashSharder(1)
	registry := &mockRegistry{}

	f := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
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
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkRedisBudgetManager_CheckAndSpend(b *testing.B) {
	redisClient := &mockRedisClient{}
	bm := NewRedisBudgetManager(redisClient, nil, time.Hour)

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	clickID := "click123"
	amount := int64(100_000)
	for b.Loop() {
		_, _ = bm.CheckAndSpend(ctx, customerID, campaignID, clickID, amount)
	}
}
