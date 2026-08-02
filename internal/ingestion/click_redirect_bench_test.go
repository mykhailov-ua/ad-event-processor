package ingestion

import (
	"testing"

	"espx/internal/config"
	"espx/internal/domain"

	"github.com/google/uuid"
)

var benchClickCampaignID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
var benchClickBrandID = uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

func benchClickHandler(b *testing.B) (*AdsPacketHandler, []byte) {
	staticCampaignMu.Lock()
	staticCampaign = &domain.Campaign{
		ID:         benchClickCampaignID,
		CustomerID: uuid.Nil,
		BrandID:    &benchClickBrandID,
		Location:   staticCampaign.Location,
	}
	staticCampaignMu.Unlock()
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			benchClickBrandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://offer.example/lp?cid={click_id}&src={sub1}",
				Weight: 100,
			}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	path := "/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&click_id=bench-click&user_id=u1&sub1=fb&gclid=GCLID99&ttclid=TTC99"
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil)
	return h, inbound
}

func BenchmarkParseClickQuery(b *testing.B) {
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&click_id=bench-click&user_id=u1&sub1=fb&gclid=GCLID99&ttclid=TTC99")
	scratch := make([]byte, 0, clickQueryScratchCap)
	parsed := &clickQueryParsed{}
	b.ReportAllocs()
	b.SetBytes(int64(len(path)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scratch = parseClickQuery(path, scratch[:0], parsed)
	}
}

func BenchmarkBuildRedirectLocation(b *testing.B) {
	base := []byte("https://offer.example/lp?cid={click_id}&src={sub1}")
	subs := [5]string{"fb"}
	pass := []byte("gclid=GCLID99&ttclid=TTC99")
	dst := make([]byte, 0, 512)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst, _ = buildRedirectLocation(dst[:0], base, "bench-click", "u1", subs, pass)
	}
}

func BenchmarkClickRedirectGnet_E2E(b *testing.B) {
	h, inbound := benchClickHandler(b)
	_, req, err := parseHTTP1(inbound, 1<<20)
	if err != nil {
		b.Fatal(err)
	}
	conn := NewGnetBenchConn(inbound)
	h.React(req, conn)
	b.ReportAllocs()
	b.SetBytes(int64(len(inbound)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.written = conn.written[:0]
		conn.responses = conn.responses[:0]
		h.React(req, conn)
	}
}

func BenchmarkClickRedirectExpandMacros(b *testing.B) {
	base := []byte("https://offer.example/lp?cid={click_id}&src={sub1}&uid={user_id}")
	subs := [5]string{"fb"}
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = expandRedirectMacros(dst[:0], base, "bench-click", "u1", subs)
	}
}
