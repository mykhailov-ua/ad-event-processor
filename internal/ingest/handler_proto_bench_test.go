package ingest

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingest/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func benchProtoHandler(b *testing.B, pbPayload *pb.AdEvent) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	handler := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	body, _ := proto.Marshal(pbPayload)
	req := parsedHTTPRequest{
		Method: []byte("POST"), Path: []byte("/track"),
		ContentType: []byte("application/x-protobuf"),
		ClientIP:    []byte("1.1.1.1"), UserAgent: []byte("Mozilla/5.0"),
		Body: body, ContentLength: len(body), HasContentLength: true,
	}
	conn := &mockGnetConn{written: make([]byte, 0, 512)}
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		handler.React(req, conn)
	}
}

func BenchmarkAdsPacketHandlerProto(b *testing.B) {
	cid := uuid.New()
	benchProtoHandler(b, &pb.AdEvent{
		CampaignId: cid[:],
		EventType:  []byte("click"),
		Metadata:   &pb.EventMetadata{ClickId: []byte("test-click")},
	})
}

func BenchmarkAdsPacketHandlerProto_ExtraBytes(b *testing.B) {
	cid := uuid.New()
	benchProtoHandler(b, &pb.AdEvent{
		CampaignId: cid[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId:    []byte("test-click"),
			UserId:     []byte("user123"),
			ExtraBytes: []byte(`{"slot":"top","cpm":"1.25"}`),
		},
	})
}

func BenchmarkAdsPacketHandlerProto_NoExtra(b *testing.B) {
	cid := uuid.New()
	benchProtoHandler(b, &pb.AdEvent{
		CampaignId: cid[:],
		EventType:  []byte("click"),
		Metadata:   &pb.EventMetadata{ClickId: []byte("test-click")},
	})
}
