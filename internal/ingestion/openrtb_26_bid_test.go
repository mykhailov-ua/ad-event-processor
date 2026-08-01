package ingestion

import (
	"testing"

	"espx/internal/config"
	"espx/internal/domain"
	"espx/internal/openrtb"
	"espx/internal/rtb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validExchangeBody() []byte {
	return []byte(`{"id":"b1","tmax":300,"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
}

func TestRunOpenRTBExchange_integration(t *testing.T) {
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
	wireReq, err := openrtb.Decode(validExchangeBody())
	require.NoError(t, err)
	out := runOpenRTBExchange(proc, wireReq, []byte("bid-1"), "8.8.8.8", openrtb.ExchangeConfig{MultiImpMax: 1})
	require.True(t, out.HasBid, "reason=%v", out.NoBid)
	assert.Equal(t, "b1", string(out.Bids[0].RequestID))
	assert.Equal(t, "1", string(out.Bids[0].ImpID))
}

func TestOpenRTBBid_gnetHandler(t *testing.T) {
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
	body := string(validExchangeBody())
	wire := BuildGnetHTTP("POST", "/openrtb/bid", map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": itoa(len(body)),
	}, []byte(body))
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.trackProc.rtbCatalog = catalog
	h.trackProc.rtbMode = rtbModeLive
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}
	_, conn := ServeGnetHarness(h, wire)
	resp := conn.Written()
	assert.Contains(t, string(resp), "200 OK")
	assert.Contains(t, string(resp), `"id":"b1"`)
	assert.Contains(t, string(resp), "x-openrtb-version: 2.6")
	assert.Contains(t, string(resp), "seatbid")
}

func TestRunOpenRTBExchange_nurlDelivery(t *testing.T) {
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
	wireReq, err := openrtb.Decode(validExchangeBody())
	require.NoError(t, err)
	exCfg := openrtb.ExchangeConfig{
		MultiImpMax:  1,
		Delivery:     openrtb.ExchangeDeliveryNURL,
		NURLTemplate: openrtb.DefaultNURLTemplate,
	}
	out := runOpenRTBExchange(proc, wireReq, []byte("bid-nurl"), "8.8.8.8", exCfg)
	require.True(t, out.HasBid)
	assert.Nil(t, out.Bids[0].AdM)
	assert.Equal(t, openrtb.DefaultNURLTemplate, out.Bids[0].NURL)
}

func TestRunOpenRTBExchange_bcatBlocksWinner(t *testing.T) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	winnerID := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: winnerID, BudgetLimit: 50_000_000}},
		map[uuid.UUID]RtbCampaignInput{
			winnerID: {BidMicro: 2_000_000, DeviceMask: 7, CategoryMask: 4, GeoHash: geo, Weight: 1},
		},
	)
	proc := trackProcessor{
		rtbCatalog: catalog,
		rtbMode:    rtbModeLive,
		ingestGeo:  &staticGeoProvider{country: "US"},
	}
	body := []byte(`{"id":"bcat-1","bcat":["IAB2"],"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	require.Equal(t, uint64(1<<2), p.BCatMask)
	var evt domain.Event
	out := runOpenRTBExchangeParsed(proc, &p.OpenRTB26Hot, &p.OpenRTB26Cold, []byte("bid-1"), "8.8.8.8", openrtb.ExchangeConfig{MultiImpMax: 1}, nil, &evt)
	assert.False(t, out.HasBid)
	assert.Equal(t, rtb.NoBidNoCandidates, out.NoBid)
}

func TestOpenRTBBid_gzipResponse(t *testing.T) {
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
	body := string(validExchangeBody())
	wire := BuildGnetHTTP("POST", "/openrtb/bid", map[string]string{
		"Content-Type":    "application/json",
		"Content-Length":  itoa(len(body)),
		"Accept-Encoding": "gzip",
	}, []byte(body))
	cfg := &config.Config{MaxRequestBodySize: 1 << 20, RtbExchangeGzip: true}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.trackProc.rtbCatalog = catalog
	h.trackProc.rtbMode = rtbModeLive
	h.trackProc.ingestGeo = &staticGeoProvider{country: "US"}
	_, conn := ServeGnetHarness(h, wire)
	resp := conn.Written()
	assert.Contains(t, string(resp), "Content-Encoding: gzip")
	assert.Contains(t, string(resp), "x-openrtb-version: 2.6")
}

func TestOpenRTBExchange_prebidIVT_rejectsAnonymous(t *testing.T) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	catalog.SetPrebidIVT(true)
	anonGeo := &staticGeoProvider{country: "US", anonymous: true}
	catalog.ConfigureRtbGates(nil, anonGeo)
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
		ingestGeo:  anonGeo,
	}
	wireReq, err := openrtb.Decode(validExchangeBody())
	require.NoError(t, err)
	out := runOpenRTBExchange(proc, wireReq, []byte("bid-1"), "8.8.8.8", openrtb.ExchangeConfig{MultiImpMax: 1})
	assert.False(t, out.HasBid)
	assert.Equal(t, rtb.NoBidPrebidIVT, out.NoBid)
}

func TestRunOpenRTBExchange_multiImp(t *testing.T) {
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
	body := []byte(`{"id":"multi-bid","tmax":300,"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}},{"id":"2","bidfloor":0.5,"banner":{"w":728,"h":90}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	require.Equal(t, uint8(2), p.ImpCount)
	var admBuf [openrtb26ImpMax][512]byte
	var evt domain.Event
	out := runOpenRTBExchangeParsed(proc, &p.OpenRTB26Hot, &p.OpenRTB26Cold, []byte("bid-multi"), "8.8.8.8", openrtb.ExchangeConfig{MultiImpMax: 10}, &admBuf, &evt)
	require.True(t, out.HasBid, "reason=%v", out.NoBid)
	assert.Equal(t, 2, out.BidCount)
	assert.Equal(t, "1", string(out.Bids[0].ImpID))
	assert.Equal(t, "2", string(out.Bids[1].ImpID))
	var buf [4096]byte
	n, err := openrtb.WriteBidsHTTPResponse(buf[:], out.ResponseWire, openrtb.HTTPWriteOpts{})
	require.NoError(t, err)
	resp := string(buf[:n])
	assert.Contains(t, resp, `"impid":"1"`)
	assert.Contains(t, resp, `"impid":"2"`)
}

func TestRunOpenRTBExchange_bseatBlocks(t *testing.T) {
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
	body := []byte(`{"id":"bseat-1","bseat":["seat-a"],"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	require.Equal(t, uint8(1), p.BSeatCount)
	var evt domain.Event
	exCfg := openrtb.ExchangeConfig{MultiImpMax: 1, SeatID: []byte("seat-a")}
	out := runOpenRTBExchangeParsed(proc, &p.OpenRTB26Hot, &p.OpenRTB26Cold, []byte("bid-1"), "8.8.8.8", exCfg, nil, &evt)
	assert.False(t, out.HasBid)
}

func TestRunOpenRTBExchange_wseatDealMatch(t *testing.T) {
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
	catalog.UpdateDeals([]rtb.DealData{{
		DealID:     "deal-ws",
		FloorMicro: 100_000,
		GeoMask:    rtb.GeoBitFromHash(geo),
		CatMask:    3,
		PacingOpen: rtb.PacingOpen,
		Seats:      2,
	}})
	proc := trackProcessor{
		rtbCatalog: catalog,
		rtbMode:    rtbModeLive,
		ingestGeo:  &staticGeoProvider{country: "US"},
	}
	body := []byte(`{"id":"wseat-1","imp":[{"id":"1","bidfloor":0.5,"pmp":{"deals":[{"id":"deal-ws","wseat":["buyer-a","buyer-b"]}]},"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	require.Equal(t, uint8(2), p.Imps[0].WSeatCount)
	var evt domain.Event
	exCfg := openrtb.ExchangeConfig{MultiImpMax: 1, SeatID: []byte("buyer-b")}
	out := runOpenRTBExchangeParsed(proc, &p.OpenRTB26Hot, &p.OpenRTB26Cold, []byte("bid-ws"), "8.8.8.8", exCfg, nil, &evt)
	require.True(t, out.HasBid, "reason=%v", out.NoBid)
	assert.Equal(t, "buyer-b", string(out.Bids[0].SeatID))

	body2 := []byte(`{"id":"wseat-2","imp":[{"id":"1","bidfloor":0.5,"pmp":{"deals":[{"id":"deal-ws","wseat":["buyer-a","buyer-b"]}]},"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
	p2 := ParseOpenRTB26(body2)
	require.True(t, p2.OK)
	out2 := runOpenRTBExchangeParsed(proc, &p2.OpenRTB26Hot, &p2.OpenRTB26Cold, []byte("bid-ws"), "8.8.8.8", openrtb.ExchangeConfig{MultiImpMax: 1, SeatID: []byte("other")}, nil, &evt)
	assert.False(t, out2.HasBid)
	assert.Equal(t, rtb.NoBidDealMismatch, out2.NoBid)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
