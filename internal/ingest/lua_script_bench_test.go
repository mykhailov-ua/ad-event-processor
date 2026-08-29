package ingest

import (
	"context"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type benchWorstRegistry struct {
	customerID uuid.UUID
	camp       *domain.Campaign
}

func (r *benchWorstRegistry) Exists(uuid.UUID) bool { return true }
func (r *benchWorstRegistry) Add(uuid.UUID, uuid.UUID, *uuid.UUID, string, domain.PacingMode, int64, string, int32, int32, []string) {
}

func (r *benchWorstRegistry) GetCustomerID(uuid.UUID) (uuid.UUID, bool) {
	if r.customerID == uuid.Nil {
		r.customerID = uuid.New()
	}
	return r.customerID, true
}

func (r *benchWorstRegistry) GetCampaign(id uuid.UUID) (*domain.Campaign, bool) {
	if r.camp != nil && r.camp.ID == id {
		return r.camp, true
	}
	custID, _ := r.GetCustomerID(id)
	cp := &domain.Campaign{
		ID:               id,
		CustomerID:       custID,
		PacingMode:       domain.PacingModeEven,
		DailyBudgetMicro: 1_000_000_000_000,
		FreqLimit:        100,
		FreqWindow:       3600,
		Location:         time.UTC,
	}
	enrichMockCampaign(cp)
	r.camp = cp
	return r.camp, true
}
func (*benchWorstRegistry) Sync(context.Context) (int, error) { return 0, nil }
func (*benchWorstRegistry) StartSync(context.Context, time.Duration) {
}
func (*benchWorstRegistry) Wait(context.Context) error { return nil }

func newLuaBenchFilter(b testing.TB, redisClient redis.UniversalClient, reg domain.CampaignRegistry, rateLimit int) *UnifiedFilter {
	b.Helper()
	f := NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
		NewJumpHashSharder(1),
		reg,
		nil,
		rateLimit,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		10_000,
	)
	f.SetLuaFastPathEnabled(true)
	f.SetFilterEvalPinWorkers(1)
	if err := f.PreloadScripts(context.Background()); err != nil {
		b.Fatal(err)
	}
	return f
}

func BenchmarkLuaScript_Happy(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(b)
	defer cleanup()

	f := newLuaBenchFilter(b, redisClient, &mockRegistry{}, 0)
	f.SetTTCMin(0)
	campID := uuid.New()
	seedCampaignBudget(b, ctx, redisClient, campID)

	payload := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"impression"}`)
	evt := &domain.Event{
		Type:       "impression",
		IP:         "203.0.113.210",
		UserID:     "bench-happy",
		CampaignID: campID,
		Payload:    payload,
	}
	setFilterDeadlineOnEvent(evt, time.Second)

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		evt.ClickID = unsafeString(strconv.AppendInt(evt.ClickIDBuf[:0], int64(benchN), 10))
		if err := f.Check(ctx, evt); err != nil {
			b.Fatal(err)
		}
		benchN++
	}
}

func BenchmarkLuaScript_Worst(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(b)
	defer cleanup()

	reg := &benchWorstRegistry{}
	f := newLuaBenchFilter(b, redisClient, reg, 100_000)
	f.SetTTCMin(500 * time.Millisecond)
	campID := uuid.New()
	seedCampaignBudget(b, ctx, redisClient, campID)

	camp, ok := reg.GetCampaign(campID)
	if !ok {
		b.Fatal("campaign setup failed")
	}
	nowMs := time.Now().UnixMilli()
	requireNoError := func(err error) {
		if err != nil {
			b.Fatal(err)
		}
	}
	requireNoError(redisClient.Set(ctx, camp.FcapKeyPrefix+"bench-worst", 0, 0).Err())
	requireNoError(redisClient.Set(ctx, camp.DailySpendKeyPrefix+time.Now().In(camp.Location).Format("20060102"), 0, 0).Err())
	var impKey []byte
	impKey = AppendCampaignHashTag(impKey[:0], campID)
	impKey = append(impKey, "imp_ts:"...)
	impKey = append(impKey, "bench-worst"...)
	impKey = append(impKey, ':')
	impKey = appendUUID(impKey, campID)
	requireNoError(redisClient.Set(ctx, string(impKey), strconv.FormatInt(nowMs, 10), time.Hour).Err())

	payload := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.211",
		UserID:     "bench-worst",
		CampaignID: campID,
		Payload:    payload,
	}
	setFilterDeadlineOnEvent(evt, 2*time.Second)

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		evt.ClickID = unsafeString(strconv.AppendInt(evt.ClickIDBuf[:0], int64(benchN), 10))
		if err := f.Check(ctx, evt); err != nil {
			b.Fatal(err)
		}
		benchN++
	}
}
