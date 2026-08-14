package ingestion

import (
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/openrtb"
	"github.com/bidshard/ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const openrtbExchangeP99Limit = 80 * time.Millisecond

func setupOpenRTBExchangeHandler(t *testing.T) *AdsPacketHandler {
	t.Helper()
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	winnerID := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: winnerID, BudgetLimit: 50_000_000}},
		map[uuid.UUID]RtbCampaignInput{
			winnerID: {BidMicro: 2_000_000, DeviceMask: 7, CategoryMask: 3, GeoHash: geo, Weight: 1},
		},
	)
	cfg := &config.Config{MaxRequestBodySize: 1 << 20, RtbExchangeMultiImpMax: 10}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.trackProc.rtbCatalog = catalog
	h.trackProc.rtbMode = rtbModeLive
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}
	return h
}

func measureOpenRTBBidLatencies(h *AdsPacketHandler, body []byte, samples int) []time.Duration {
	wire := BuildGnetHTTP("POST", "/openrtb/bid", map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": itoa(len(body)),
	}, body)
	latencies := make([]time.Duration, 0, samples)
	for range samples {
		start := time.Now()
		_, conn := ServeGnetHarness(h, wire)
		_ = conn.Written()
		latencies = append(latencies, time.Since(start))
	}
	return latencies
}

func TestOpenRTB26_Exchange_LatencySLA(t *testing.T) {
	h := setupOpenRTBExchangeHandler(t)
	body := validExchangeBody()
	const samples = 256
	latencies := measureOpenRTBBidLatencies(h, body, samples)
	require.NotEmpty(t, latencies)

	p99 := percentileDuration(latencies, 99)
	t.Logf("openrtb26 exchange gnet n=%d p50=%v p99=%v", samples, percentileDuration(latencies, 50), p99)
	require.Less(t, p99, openrtbExchangeP99Limit, "/openrtb/bid handler p99 must stay <80 ms")
}

func TestOpenRTB26_Exchange_Core_LatencySLA(t *testing.T) {
	h := setupOpenRTBExchangeHandler(t)
	body := validExchangeBody()
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	p.Flags |= openrtb26FlagTest
	proc := h.trackProc
	var admBuf [openrtb26ImpMax][512]byte
	exCfg := openrtb.ExchangeConfig{MultiImpMax: 10, SeatID: []byte("1")}

	const samples = 512
	latencies := make([]time.Duration, 0, samples)
	for range samples {
		var evt domain.Event
		start := time.Now()
		out := runOpenRTBExchangeParsed(proc, &p.OpenRTB26Hot, &p.OpenRTB26Cold, []byte("bid-sla"), "8.8.8.8", exCfg, &admBuf, &evt)
		latencies = append(latencies, time.Since(start))
		require.True(t, out.HasBid)
	}
	p99 := percentileDuration(latencies, 99)
	t.Logf("openrtb26 exchange core n=%d p99=%v", samples, p99)
	require.Less(t, p99, openrtbExchangeP99Limit)
}

func BenchmarkOpenRTB26_exchangeGnet(b *testing.B) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	winnerID := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: winnerID, BudgetLimit: 50_000_000}},
		map[uuid.UUID]RtbCampaignInput{
			winnerID: {BidMicro: 2_000_000, DeviceMask: 7, CategoryMask: 3, GeoHash: geo, Weight: 1},
		},
	)
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.trackProc.rtbCatalog = catalog
	h.trackProc.rtbMode = rtbModeLive
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}
	body := validExchangeBody()
	wire := BuildGnetHTTP("POST", "/openrtb/bid", map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": itoa(len(body)),
	}, body)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, conn := ServeGnetHarness(h, wire)
		_ = conn.Written()
	}
}

func BenchmarkRunOpenRTBExchangeParsed(b *testing.B) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	winnerID := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: winnerID, BudgetLimit: 50_000_000}},
		map[uuid.UUID]RtbCampaignInput{
			winnerID: {BidMicro: 2_000_000, DeviceMask: 7, CategoryMask: 3, GeoHash: geo, Weight: 1},
		},
	)
	proc := trackProcessor{
		rtbCatalog: catalog,
		rtbMode:    rtbModeLive,
		ingestGeo:  &staticGeoProvider{country: "US"},
	}
	body := validExchangeBody()
	p := ParseOpenRTB26(body)
	exCfg := openrtb.ExchangeConfig{MultiImpMax: 1, SeatID: []byte("1")}
	var admBuf [openrtb26ImpMax][512]byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var evt domain.Event
		_ = runOpenRTBExchangeParsed(proc, &p.OpenRTB26Hot, &p.OpenRTB26Cold, []byte("bid-bench"), "8.8.8.8", exCfg, &admBuf, &evt)
	}
}
