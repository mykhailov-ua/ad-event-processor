package ingestion

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion/pb"
	"github.com/google/uuid"
)

func buildProtoTrackPayload(b *testing.B) []byte {
	b.Helper()
	id := uuid.New()
	pbPayload := &pb.AdEvent{
		CampaignId: id[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId: []byte("test-click"),
		},
	}
	body, err := pbPayload.MarshalVT()
	if err != nil {
		b.Fatal(err)
	}
	return body
}

func BenchmarkHotPath_timeNow(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = time.Now()
	}
}

func BenchmarkHotPath_monotonicNano(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = monotonicNano()
	}
}

func BenchmarkHotPath_cachedTimeUTC(b *testing.B) {
	storeCachedNowUTC()
	b.ReportAllocs()
	for b.Loop() {
		_ = CachedTimeUTC()
	}
}

func BenchmarkHotPath_filterEngineDeadlineCheck(b *testing.B) {
	ctx := attachFilterDeadline(context.Background(), 5*time.Second)
	b.ReportAllocs()
	for b.Loop() {
		_ = filterDeadlineExceeded(ctx)
	}
}

func BenchmarkHotPath_filterEngineCheck_noTimeout(b *testing.B) {
	engine := NewFilterEngine(0, &countingFilter{})
	evt := &domain.Event{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = engine.Check(ctx, evt)
	}
}

func BenchmarkHotPath_filterEngineCheck_withDeadline(b *testing.B) {
	engine := NewFilterEngine(5*time.Second, &countingFilter{})
	evt := &domain.Event{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = engine.Check(ctx, evt)
	}
}

func BenchmarkHotPath_latencyRingRecord(b *testing.B) {
	ring := NewLatencyRing(defaultLatencyRingCap)
	start := monotonicNano()
	b.ReportAllocs()
	for b.Loop() {
		ring.RecordMono(start)
	}
}

func BenchmarkHotPath_counterInc(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		filterGeoLookupErrors.Inc()
	}
}

func BenchmarkHotPath_handlerProto_delegate(b *testing.B) {
	BenchmarkAdsPacketHandlerProto(b)
}

func BenchmarkHotPath_AdsPacketHandlerProto_reject404(b *testing.B) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	registry := &mockRegistry{}
	sharder := NewJumpHashSharder(1)
	handler := NewAdsPacketHandler(cfg, registry, NewFilterEngine(0, &errFilter{err: ErrCampaignNotFound}), nil, nil, sharder, "fraud-stream", nil)

	payload := buildProtoTrackPayload(b)
	req := parsedHTTPRequest{
		Method:           []byte("POST"),
		Path:             []byte("/track"),
		ContentType:      []byte("application/x-protobuf"),
		Body:             payload,
		ContentLength:    len(payload),
		HasContentLength: true,
	}
	conn := &mockGnetConn{written: make([]byte, 0, 512)}

	b.ReportAllocs()
	for b.Loop() {
		handler.React(req, conn)
	}
}

func BenchmarkHotPath_AdsPacketHandlerProto_infra503(b *testing.B) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	handler := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, infraErrFilter{}), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	payload := buildProtoTrackPayload(b)
	req := parsedHTTPRequest{
		Method:           []byte("POST"),
		Path:             []byte("/track"),
		ContentType:      []byte("application/x-protobuf"),
		Body:             payload,
		ContentLength:    len(payload),
		HasContentLength: true,
	}
	conn := &mockGnetConn{written: make([]byte, 0, 512)}

	b.ReportAllocs()
	for b.Loop() {
		handler.React(req, conn)
	}
}
