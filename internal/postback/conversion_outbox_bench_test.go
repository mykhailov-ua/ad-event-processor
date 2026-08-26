package postback

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var benchConversionPayload = []byte(`{
	"sub1":"fb","sub5":"kw","sub30":"deep",
	"fbclid":"fb-abc","gclid":"gc-xyz","ttclid":"tt-99",
	"email":"user@example.com","payout_micro":1500000
}`)

var (
	benchCampaignID = uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	benchCustomerID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	benchConvEvent  = &domain.Event{
		ClickID:            "click-bench-1234567890",
		CampaignID:         benchCampaignID,
		UserID:             "user-1",
		Type:               "conversion",
		Payload:            benchConversionPayload,
		ClearingPriceMicro: 2_500_000,
	}
)

func BenchmarkBuildPostbackPayloadFromEvent(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = buildPostbackPayloadFromEvent(benchConvEvent, benchCustomerID)
	}
}

func BenchmarkMarshalPostbackPayload(b *testing.B) {
	pb := buildPostbackPayloadFromEvent(benchConvEvent, benchCustomerID)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(pb)
	}
}

func BenchmarkMergeEventPayloadInto(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var pb PostbackPayload
		mergeEventPayloadInto(&pb, benchConversionPayload)
	}
}

func BenchmarkEventTypeMatches(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if !eventTypeMatches("conversion", "conversion") {
			b.Fatal()
		}
	}
}

type benchPostbackQuerier struct {
	hasConfig   bool
	outboxCalls int
}

func (q *benchPostbackQuerier) ListPostbackConfigsByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.PostbackConfig, error) {
	if !q.hasConfig {
		return nil, nil
	}
	out := make([]db.PostbackConfig, 0, len(ids))
	for _, id := range ids {
		out = append(out, db.PostbackConfig{
			CampaignID:  id,
			Provider:    "facebook",
			UrlTemplate: "1234567890",
			TargetEvent: "conversion",
		})
	}
	return out, nil
}

func (q *benchPostbackQuerier) ListCampaignsByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.Campaign, error) {
	out := make([]db.Campaign, 0, len(ids))
	for _, id := range ids {
		out = append(out, db.Campaign{
			ID:         id,
			CustomerID: pgtype.UUID{Bytes: benchCustomerID, Valid: true},
		})
	}
	return out, nil
}

func (q *benchPostbackQuerier) CreateOutboxEventsBatch(ctx context.Context, arg db.CreateOutboxEventsBatchParams) error {
	q.outboxCalls++
	return nil
}

func (q *benchPostbackQuerier) GetPostbackConfig(ctx context.Context, id pgtype.UUID) (db.PostbackConfig, error) {
	if !q.hasConfig {
		return db.PostbackConfig{}, pgx.ErrNoRows
	}
	return db.PostbackConfig{
		CampaignID:  id,
		Provider:    "facebook",
		UrlTemplate: "1234567890",
		TargetEvent: "conversion",
	}, nil
}

func (q *benchPostbackQuerier) GetCampaign(ctx context.Context, id pgtype.UUID) (db.Campaign, error) {
	return db.Campaign{
		ID:         id,
		CustomerID: pgtype.UUID{Bytes: benchCustomerID, Valid: true},
	}, nil
}

func (q *benchPostbackQuerier) CreateOutboxEvent(ctx context.Context, arg db.CreateOutboxEventParams) (db.OutboxEvent, error) {
	return db.OutboxEvent{ID: 1, EventType: arg.EventType, Payload: arg.Payload}, nil
}

func BenchmarkConversionEnqueue_skipNoConfig(b *testing.B) {
	enq := NewConversionPostbackEnqueuer(&benchPostbackQuerier{hasConfig: false})
	events := []*domain.Event{benchConvEvent}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		enq.OnBatchStored(ctx, events)
	}
}

func BenchmarkConversionEnqueue_fullCPU(b *testing.B) {
	enq := NewConversionPostbackEnqueuer(&benchPostbackQuerier{hasConfig: true})
	events := []*domain.Event{benchConvEvent}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		enq.OnBatchStored(ctx, events)
	}
}
