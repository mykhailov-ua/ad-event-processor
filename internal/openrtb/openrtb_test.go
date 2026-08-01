package openrtb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeEncodeRoundtrip(t *testing.T) {
	raw := []byte(`{
  "id": "req-golden-001",
  "tmax": 250,
  "test": 0,
  "cur": ["USD"],
  "imp": [{
    "id": "imp-1",
    "bidfloor": 1.25,
    "bidfloorcur": "USD",
    "banner": {"w": 300, "h": 250}
  }],
  "site": {"domain": "example.com", "page": "https://example.com/"},
  "device": {"ip": "203.0.113.1", "ua": "Mozilla/5.0", "devicetype": 2, "geo": {"country": "USA"}},
  "user": {"id": "u1"}
}`)
	req, err := Decode(raw)
	require.NoError(t, err)
	assert.Equal(t, "req-golden-001", req.ID)
	assert.Len(t, req.Imp, 1)
	assert.Equal(t, "imp-1", req.Imp[0].ID)
	assert.InDelta(t, 1.25, req.Imp[0].BidFloor, 0.001)

	vr := Validate(req, ExchangeConfig{MultiImpMax: 1})
	assert.True(t, vr.Valid, vr.Errors)

	resp := BidResponse{
		ID:    req.ID,
		BidID: "bid-abc",
		Cur:   "USD",
		SeatBid: []SeatBid{{
			Bid: []Bid{{
				ID:    "bid-abc",
				ImpID: "imp-1",
				Price: 1.50,
				AdM:   "<html>ad</html>",
				AdID:  "ad1",
				CrID:  "cr1",
				CID:   "camp1",
			}},
		}},
	}
	wire, err := EncodeBid(resp)
	require.NoError(t, err)
	assert.Contains(t, string(wire), `"id":"req-golden-001"`)
	assert.Contains(t, string(wire), `"impid":"imp-1"`)
}

func TestValidate_missingInventory(t *testing.T) {
	raw := []byte(`{"id":"x","imp":[{"id":"1","bidfloor":0.5,"banner":{"w":1,"h":1}}],"device":{"ip":"1.1.1.1","ua":"x"}}`)
	req, err := Decode(raw)
	require.NoError(t, err)
	vr := Validate(req, ExchangeConfig{MultiImpMax: 1})
	assert.False(t, vr.Valid)
	assert.Contains(t, vr.Errors[0], "site or app")
}

func TestValidate_rejectsOpenRTB30(t *testing.T) {
	raw := []byte(`{"openrtb":{"request":{"id":"r1","item":[{"id":"1"}]}}}`)
	_, err := Decode(raw)
	assert.ErrorIs(t, err, ErrOpenRTB30)
}

func TestWriteBidHTTPResponse_singlePass(t *testing.T) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "b1")
	copy(bidID[:], "bid-abc")
	copy(impID[:], "1")
	copy(camp[:], "00000000-0000-0000-0000-000000000001")
	var buf [2048]byte
	n, err := WriteBidHTTPResponse(buf[:], BidWire{
		RequestID:  reqID[:2],
		BidID:      bidID[:len("bid-abc")],
		ImpID:      impID[:1],
		PriceMicro: 2_000_000,
		CurUSD:     true,
		AdM:        []byte(`<html>ad</html>`),
		CampaignID: camp[:36],
		CreativeID: 7,
	}, HTTPWriteOpts{})
	require.NoError(t, err)
	resp := string(buf[:n])
	assert.True(t, len(resp) > BidHTTPHdrSize)
	assert.Contains(t, resp, "HTTP/1.1 200 OK")
	assert.Contains(t, resp, "Content-Length:")
	assert.Contains(t, resp, `"id":"b1"`)
	assert.Contains(t, resp, `"price":2.000000`)
	// JSON starts immediately after fixed header - no gap, no second copy.
	assert.Equal(t, byte('{'), buf[BidHTTPHdrSize])
}

func TestAppendBidResponse_zeroAllocShape(t *testing.T) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "req-golden-001")
	copy(bidID[:], "bid-abc")
	copy(impID[:], "imp-1")
	copy(camp[:], "camp-uuid-0001-0002-0003-0004")
	adm := []byte(`<html>ad</html>`)
	var buf [1024]byte
	wire, err := AppendBidResponse(buf[:0], BidWire{
		RequestID:  reqID[:len("req-golden-001")],
		BidID:      bidID[:len("bid-abc")],
		ImpID:      impID[:len("imp-1")],
		PriceMicro: 1_500_000,
		CurUSD:     true,
		AdM:        adm,
		CampaignID: camp[:len("camp-uuid-0001-0002-0003-0004")],
		CreativeID: 42,
	})
	require.NoError(t, err)
	s := string(wire)
	assert.Contains(t, s, `"id":"req-golden-001"`)
	assert.Contains(t, s, `"impid":"imp-1"`)
	assert.Contains(t, s, `"price":1.500000`)
	assert.Contains(t, s, `"crid":"42"`)
}

func TestAppendBidResponse_nurl(t *testing.T) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "req-nurl")
	copy(bidID[:], "bid-nurl")
	copy(impID[:], "1")
	copy(camp[:], "camp-uuid-0001-0002-0003-0004")
	nurl := []byte("https://win.example/n?price=${AUCTION_PRICE}")
	var buf [1024]byte
	wire, err := AppendBidResponse(buf[:0], BidWire{
		RequestID:  reqID[:len("req-nurl")],
		BidID:      bidID[:len("bid-nurl")],
		ImpID:      impID[:1],
		PriceMicro: 1_000_000,
		CurUSD:     true,
		NURL:       nurl,
		CampaignID: camp[:len("camp-uuid-0001-0002-0003-0004")],
		CreativeID: 9,
	})
	require.NoError(t, err)
	s := string(wire)
	assert.Contains(t, s, `"nurl":"https://win.example/n?price=${AUCTION_PRICE}"`)
	assert.NotContains(t, s, `"adm"`)
}

func TestAppendBidResponseWire_multi(t *testing.T) {
	var reqID, bidID, imp1, imp2, camp [36]byte
	copy(reqID[:], "multi-req")
	copy(bidID[:], "bid-multi")
	copy(imp1[:], "1")
	copy(imp2[:], "2")
	copy(camp[:], "camp-uuid-0001-0002-0003-0004")
	adm := []byte(`<html>ad</html>`)
	var buf [2048]byte
	wire, err := AppendBidResponseWire(buf[:0], BidResponseWire{
		RequestID: reqID[:len("multi-req")],
		BidID:     bidID[:len("bid-multi")],
		CurUSD:    true,
		Bids: []BidWire{
			{ImpID: imp1[:1], PriceMicro: 1_000_000, AdM: adm, CampaignID: camp[:len("camp-uuid-0001-0002-0003-0004")], CreativeID: 1},
			{ImpID: imp2[:1], PriceMicro: 2_000_000, AdM: adm, CampaignID: camp[:len("camp-uuid-0001-0002-0003-0004")], CreativeID: 2},
		},
	})
	require.NoError(t, err)
	s := string(wire)
	assert.Contains(t, s, `"impid":"1"`)
	assert.Contains(t, s, `"impid":"2"`)
	assert.Contains(t, s, `"bidid":"bid-multi"`)
}

func TestWriteBidHTTPResponse_gzip(t *testing.T) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "gzip-req-001")
	copy(bidID[:], "bid-gzip")
	copy(impID[:], "imp-1")
	copy(camp[:], "00000000-0000-0000-0000-000000000001")
	adm := make([]byte, 256)
	for i := range adm {
		adm[i] = 'x'
	}
	var buf [4096]byte
	n, err := WriteBidHTTPResponse(buf[:], BidWire{
		RequestID:  reqID[:len("gzip-req-001")],
		BidID:      bidID[:len("bid-gzip")],
		ImpID:      impID[:len("imp-1")],
		PriceMicro: 3_000_000,
		CurUSD:     true,
		AdM:        adm,
		CampaignID: camp[:36],
		CreativeID: 11,
	}, HTTPWriteOpts{Gzip: true})
	require.NoError(t, err)
	resp := string(buf[:n])
	assert.Contains(t, resp, "Content-Encoding: gzip")
	assert.Contains(t, resp, "x-openrtb-version: 2.6")
}

func TestAppendNoBidResponse(t *testing.T) {
	var buf [128]byte
	wire := AppendNoBidResponse(buf[:0], []byte("req-1"), 2)
	assert.Contains(t, string(wire), `"id":"req-1"`)
	assert.Contains(t, string(wire), `"nbr":2`)
}

func TestEncodeNoBid(t *testing.T) {
	wire, err := EncodeNoBid("req-1", 2)
	require.NoError(t, err)
	assert.Contains(t, string(wire), `"id":"req-1"`)
	assert.Contains(t, string(wire), `"nbr":2`)
}

func TestApplyMacros_P0(t *testing.T) {
	template := []byte("https://win.example/n?price=${AUCTION_PRICE}&id=${AUCTION_ID}&bid=${AUCTION_BID_ID}&imp=${AUCTION_IMP_ID}&seat=${AUCTION_SEAT_ID}")
	var buf [512]byte
	got := string(AppendApplyMacros(buf[:0], template, MacroWire{
		AuctionPrice: []byte("1.230000"),
		AuctionID:    []byte("req-1"),
		BidID:        []byte("bid-1"),
		ImpID:        []byte("imp-1"),
		SeatID:       []byte("seat-1"),
	}))
	assert.Contains(t, got, "price=1.230000")
	assert.Contains(t, got, "id=req-1")
	assert.Contains(t, got, "bid=bid-1")
	assert.Contains(t, got, "imp=imp-1")
	assert.Contains(t, got, "seat=seat-1")

	gotCold := ApplyMacros(string(template), MacroContext{
		AuctionPrice: "1.230000",
		AuctionID:    "req-1",
		BidID:        "bid-1",
		ImpID:        "imp-1",
		SeatID:       "seat-1",
	})
	assert.Equal(t, got, gotCold)
}

func TestIntegrationProfile(t *testing.T) {
	p := Profile()
	assert.Equal(t, "2.6", p.OpenRTBVersion)
	assert.Contains(t, p.Required, "id")
	assert.Contains(t, p.NotSupported, "imp.native")
}

func TestDecodeGoldenFixtureFile(t *testing.T) {
	raw, err := os.ReadFile("testdata/bid_request_min.json")
	require.NoError(t, err)
	req, err := Decode(raw)
	require.NoError(t, err)
	assert.Equal(t, "req-golden-001", req.ID)
	vr := Validate(req, ExchangeConfig{MultiImpMax: 1})
	assert.True(t, vr.Valid, vr.Errors)
}
