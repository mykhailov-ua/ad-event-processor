package ingest

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

var (
	benchClickCampaignID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	benchClickBrandID    = uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
)

func benchClickHandler(b *testing.B) (*AdsPacketHandler, []byte) {
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:         benchClickCampaignID,
			CustomerID: uuid.Nil,
			BrandID:    &benchClickBrandID,
			Location:   (*campPtr).Location,
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
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
	for b.Loop() {
		scratch = parseClickQuery(path, scratch[:0], parsed)
	}
}

func BenchmarkParseClickQuery30Params(b *testing.B) {
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&click_id=bench-click&user_id=u1" +
		"&sub1=a&sub2=b&sub3=c&sub4=d&sub5=e&sub6=f&sub7=g&sub8=h&sub9=i&sub10=j" +
		"&sub11=k&sub12=l&sub13=m&sub14=n&sub15=o&sub16=p&sub17=q&sub18=r&sub19=s&sub20=t" +
		"&sub21=u&sub22=v&sub23=w&sub24=x&sub25=y&sub26=z&sub27=aa&sub28=bb&sub29=cc&sub30=dd" +
		"&gclid=GCLID99&ttclid=TTC99&fbclid=FBC99")
	scratch := make([]byte, 0, clickQueryScratchCap)
	parsed := &clickQueryParsed{}
	b.ReportAllocs()
	b.SetBytes(int64(len(path)))
	for b.Loop() {
		scratch = parseClickQuery(path, scratch[:0], parsed)
	}
}

func BenchmarkBuildRedirectLocation(b *testing.B) {
	base := []byte("https://offer.example/lp?cid={click_id}&src={sub1}")
	subs := SubIDSlots{"fb"}
	pass := []byte("gclid=GCLID99&ttclid=TTC99")
	dst := make([]byte, 0, 512)
	b.ReportAllocs()
	for b.Loop() {
		dst, _ = buildRedirectLocation(dst[:0], base, "bench-click", "u1", subs, pass)
	}
}

func BenchmarkClickRedirectGnet_E2E(b *testing.B) {
	h, inbound := benchClickHandler(b)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	if err != nil {
		b.Fatal(err)
	}
	conn := NewGnetBenchConn(inbound)
	h.React(req, conn)
	b.ReportAllocs()
	b.SetBytes(int64(len(inbound)))
	for b.Loop() {
		conn.written = conn.written[:0]
		conn.responses = conn.responses[:0]
		h.React(req, conn)
	}
}

func BenchmarkClickRedirectExpandMacros(b *testing.B) {
	base := []byte("https://offer.example/lp?cid={click_id}&src={sub1}&uid={user_id}")
	subs := SubIDSlots{"fb"}
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	for b.Loop() {
		dst = expandRedirectMacros(dst[:0], base, "bench-click", "u1", subs)
	}
}
