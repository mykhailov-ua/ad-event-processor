package ingestion

import (
	"encoding/json"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

func benchSafePageCampaign(b *testing.B) (*AdsPacketHandler, uuid.UUID) {
	cid := benchClickCampaignID
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &benchClickBrandID
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			benchClickBrandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://money.example/lp?cid={click_id}",
				Weight: 100,
			}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &countingFilter{})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	return h, cid
}

func BenchmarkClickRedirectGnet_forceSafe(b *testing.B) {
	h, cid := benchSafePageCampaign(b)
	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=bench-safe&user_id=u1&gclid=GCLID99"
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":                      "keep-alive",
		"Content-Length":                  "0",
		"User-Agent":                      "Mozilla/5.0",
		"X-ad-event-processor-Force-Safe": "1",
	}, nil)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
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

func BenchmarkSafePageStubGnet_E2E(b *testing.B) {
	h, cid := benchSafePageCampaign(b)
	path := "/safe_page_stub?campaign_id=" + cid.String()
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
	if err != nil {
		b.Fatal(err)
	}
	conn := NewGnetBenchConn(inbound)
	h.React(req, conn)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.written = conn.written[:0]
		conn.responses = conn.responses[:0]
		h.React(req, conn)
	}
}

func BenchmarkTrackVerifyGnet_E2E(b *testing.B) {
	h, cid := benchSafePageCampaign(b)
	events := humanMouseEvents(15)
	events = append(events,
		safePageVerifyEvent{T: "touchstart", TS: 100},
		safePageVerifyEvent{T: "scroll", TS: 101},
	)
	body, err := json.Marshal(safePageVerifyRequest{
		CampaignID:  cid.String(),
		Events:      events,
		Fingerprint: validAdvancedFingerprint(),
	})
	if err != nil {
		b.Fatal(err)
	}
	inbound := BuildGnetHTTP("POST", safePageVerifyPath, map[string]string{
		"Content-Type": "application/json",
		"Connection":   "keep-alive",
	}, body)
	_, req, err := parseHTTP1(inbound, 1<<20, nil)
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

func BenchmarkResolveSafePageAction(b *testing.B) {
	id := benchClickCampaignID
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID:              id,
			SafePageURL:     "https://safe.example/lp",
			SafePageEnabled: true,
		},
	}
	out := trackOutcome{Status: trackStatusFraudAccepted, RejectKind: filterRejectFraud}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolveSafePageAction(reg, id, out, false)
	}
}
