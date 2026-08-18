package ingestion

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"
	"github.com/bidshard/ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type benchTrackRegistry struct {
	mockRegistry
	license licensing.LicenseState
}

func (r *benchTrackRegistry) GetLicenseState() (licensing.LicenseState, licensing.Entitlements) {
	return r.license, licensing.Entitlements{}
}

func buildTrackE2EAcceptHandler(b *testing.B, campID uuid.UUID, rdb redis.UniversalClient) *AdsPacketHandler {
	b.Helper()
	ctx := context.Background()
	registry := &benchTrackRegistry{license: licensing.StateActive}
	cp, ok := registry.GetCampaign(campID)
	if !ok {
		b.Fatal("campaign missing")
	}
	cachedMockCamp.Store(cp)
	seedCampaignBudget(b, ctx, rdb, campID)

	rdbs := []redis.UniversalClient{rdb}
	sharder := NewJumpHashSharder(1)
	unified := NewUnifiedFilter(
		rdbs,
		sharder,
		registry,
		nil,
		10_000,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		10_000,
	)
	unified.SetLuaFastPathEnabled(true)
	unified.SetFilterEvalPinWorkers(1)
	if err := unified.PreloadScripts(ctx); err != nil {
		b.Fatal(err)
	}
	engine := NewFilterEngine(2*time.Second, NewLicenseFilter(registry), unified)

	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	return NewAdsPacketHandler(cfg, registry, engine, nil, rdbs, sharder, "fraud-stream", nil)
}

func BenchmarkTrackE2E_accept(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}
	rdb, cleanup := setupTestRedis(b)
	defer cleanup()

	campID := uuid.New()
	handler := buildTrackE2EAcceptHandler(b, campID, rdb)

	pbPayload := buildProtoTrackPayloadWithCampaign(b, campID)
	body, err := pbPayload.MarshalVT()
	if err != nil {
		b.Fatal(err)
	}
	req := parsedHTTPRequest{
		Method:           []byte("POST"),
		Path:             []byte("/track"),
		ContentType:      []byte("application/x-protobuf"),
		ClientIP:         []byte("1.1.1.1"),
		UserAgent:        []byte("Mozilla/5.0"),
		Body:             body,
		ContentLength:    len(body),
		HasContentLength: true,
	}
	conn := &mockGnetConn{written: make([]byte, 0, 512)}

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pbPayload.Metadata.UserId = []byte("bench-" + strconv.Itoa(i))
		body, err := pbPayload.MarshalVT()
		if err != nil {
			b.Fatal(err)
		}
		req.Body = body
		req.ContentLength = len(body)
		handler.React(req, conn)
	}
}

func buildProtoTrackPayloadWithCampaign(b *testing.B, campID uuid.UUID) *pb.AdEvent {
	b.Helper()
	return &pb.AdEvent{
		CampaignId: campID[:],
		EventType:  []byte("impression"),
		Metadata: &pb.EventMetadata{
			ClickId: []byte("bench-impression"),
			UserId:  []byte("bench-0"),
		},
	}
}
