package ingestion

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func benchTgClickHandler(b testing.TB) (*AdsPacketHandler, parsedHTTPRequest, *GnetHarnessConn) {
	staticCampaignMu.Lock()
	staticCampaign = &domain.Campaign{
		ID:         benchClickCampaignID,
		CustomerID: uuid.Nil,
		BrandID:    &benchClickBrandID,
		Location:   staticCampaign.Location,
	}
	staticCampaignMu.Unlock()
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			benchClickBrandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://offer.example/lp?cid={click_id}&token={bridge_token}",
				Weight: 100,
			}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	path := "/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=token_abc123_&premium=1&motivated=true&sub1=testsub"
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0 Telegram-Android",
	}, nil)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	if err != nil {
		b.Fatal(err)
	}
	conn := NewGnetBenchConn(inbound)
	h.React(req, conn)
	return h, req, conn
}

func TestTgClickRedirectGnet_ZeroAlloc(t *testing.T) {
	h, req, conn := benchTgClickHandler(t)
	allocs := testing.AllocsPerRun(100, func() {
		conn.written = conn.written[:0]
		conn.responses = conn.responses[:0]
		h.React(req, conn)
	})
	if allocs != 0.0 {
		t.Fatalf("expected 0 allocs/op, got %v", allocs)
	}
}

func BenchmarkTgClickRedirectGnet_E2E(b *testing.B) {
	h, req, conn := benchTgClickHandler(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(req.Path)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.written = conn.written[:0]
		conn.responses = conn.responses[:0]
		h.React(req, conn)
	}
}

func BenchmarkBuildTgRedirectLocation(b *testing.B) {
	base := []byte("https://offer.example/lp?cid={click_id}&token={bridge_token}&src={sub1}")
	subs := [5]string{"fb"}
	dst := make([]byte, 0, 512)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst, _ = buildTgRedirectLocation(dst[:0], base, "d5671191-236b-4e94-825e-399185a9bc8d", "bridge_abc123", subs, nil)
	}
}
