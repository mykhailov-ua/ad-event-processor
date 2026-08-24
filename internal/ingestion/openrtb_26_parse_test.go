package ingestion

import (
	"testing"

	"ad-event-processor/internal/openrtb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var openrtb26Sample = []byte(`{
  "id":"req-1",
  "tmax":250,
  "imp":[{"id":"1","bidfloor":1.25,"pmp":{"deals":[{"id":"deal-a"}]}}],
  "device":{"devicetype":1},
  "site":{"cat":["IAB1"]}
}`)

func TestParseOpenRTB26_fields(t *testing.T) {
	p := ParseOpenRTB26(openrtb26Sample)
	require.True(t, p.OK)
	assert.Equal(t, int64(1_250_000), p.BidFloorMicro)
	assert.Equal(t, uint8(2), p.DeviceType)
	assert.Equal(t, int32(250), p.TmaxMs)
	assert.Equal(t, uint8(1), p.SeatCount)
	assert.Equal(t, "deal-a", string(p.DealID[:p.DealIDLen]))
	assert.Equal(t, "req-1", string(p.RequestID[:p.RequestIDLen]))
	assert.Equal(t, "1", string(p.ImpID[:p.ImpIDLen]))
	assert.Equal(t, uint8(1), p.ImpCount)
	assert.True(t, p.Flags&openrtb26FlagSite != 0)
}

func TestParseOpenRTB26_audienceFields(t *testing.T) {
	body := []byte(`{"id":"r1","imp":[{"id":"1","bidfloor":1.0,"secure":1,"banner":{"w":300,"h":250}}],"app":{"bundle":"com.example.app","cat":["IAB2"]},"device":{"ip":"1.1.1.1","ua":"Mozilla","os":"Android","language":"en","geo":{"country":"US","region":"CA"}},"user":{"id":"uid-1","buyeruid":"dsp-cookie-9"},"regs":{"coppa":0}}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	assert.Equal(t, "com.example.app", string(p.AppBundle[:p.AppBundleLen]))
	assert.Equal(t, "Android", string(p.DeviceOS[:p.DeviceOSLen]))
	assert.Equal(t, "en", string(p.DeviceLang[:p.DeviceLangLen]))
	assert.Equal(t, "CA", string(p.GeoRegion[:p.GeoRegionLen]))
	assert.Equal(t, "dsp-cookie-9", string(p.BuyerUID[:p.BuyerUIDLen]))
	assert.Equal(t, uint16(300), p.BannerW)
	assert.Equal(t, uint16(250), p.BannerH)
	assert.True(t, p.Flags&openrtb26FlagSecure != 0)
	assert.NotZero(t, p.FcapUserHash)
	_ = mapParsedToTargeting(&p.OpenRTB26Hot, &p.OpenRTB26Cold, nil, "")
}

func TestParseOpenRTB26_extensionFields(t *testing.T) {
	body := []byte(`{
	  "id":"ext-1",
	  "source":{"tid":"supply-txn-42"},
	  "imp":[{"id":"1","bidfloor":1.0,"secure":1,"pmp":{"private":1,"deals":[{"id":"d1"}]},
	    "metric":[{"type":"viewability","value":0.85,"vendor":"EXCHANGE"}],
	    "banner":{"w":728,"h":90}}],
	  "site":{"domain":"news.example","page":"https://news.example/article"},
	  "device":{"ip":"9.9.9.9","ua":"Mozilla/5.0","ifa":"AEBE52E7-03EE-455A-9553-811439C4D3BB","lmt":0,"connectiontype":2},
	  "app":{"bundle":"com.foo","ver":"3.2.1"},
	  "user":{"ext":{"eids":[{"source":"uidapi.com","uids":[{"id":"uid-abc","atype":1}]}]}}
	}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	assert.Equal(t, "supply-txn-42", string(p.SourceTID[:p.SourceTIDLen]))
	assert.Equal(t, uint8(1), p.PMPPrivate)
	assert.Equal(t, uint8(2), p.ConnectionType)
	assert.Equal(t, "AEBE52E7-03EE-455A-9553-811439C4D3BB", string(p.DeviceIFA[:p.DeviceIFALen]))
	assert.Equal(t, "https://news.example/article", string(p.SitePage[:p.SitePageLen]))
	assert.Equal(t, "3.2.1", string(p.AppVer[:p.AppVerLen]))
	assert.Equal(t, "uidapi.com", string(p.EIDSource[:p.EIDSourceLen]))
	assert.Equal(t, "uid-abc", string(p.EIDUID[:p.EIDUIDLen]))
	assert.Equal(t, uint8(1), p.EIDCount)
	assert.Equal(t, "viewability", string(p.MetricType[:p.MetricTypeLen]))
	assert.Equal(t, uint32(850_000), p.MetricValuePPM)
	targeting := mapParsedToTargeting(&p.OpenRTB26Hot, &p.OpenRTB26Cold, nil, "")
	assert.NotZero(t, p.FcapUserHash)
	assert.Equal(t, uint8(2), targeting.Input.ConnectionType)
	assert.Equal(t, uint32(850_000), targeting.Input.ViewabilityPPM)
	assert.Equal(t, uint8(1), targeting.Input.PMPPrivate)
}

func TestParseOpenRTB26_blocklists(t *testing.T) {
	body := []byte(`{
	  "id":"blk-1",
	  "bcat":["IAB1","IAB2-3"],
	  "badv":["evil.example","bidshard.local"],
	  "bapp":["com.blocked"],
	  "imp":[{"id":"1","bidfloor":1.0,"banner":{"w":300,"h":250}}],
	  "app":{"bundle":"com.blocked"},
	  "device":{"ip":"1.2.3.4","ua":"Mozilla/5.0"}
	}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	assert.Equal(t, uint8(2), p.BCatCount)
	assert.Equal(t, "IAB1", string(p.BCat[0][:p.BCatLen[0]]))
	assert.Equal(t, uint8(2), p.BAdvCount)
	assert.Equal(t, uint8(1), p.BAppCount)
	assert.Equal(t, uint64(1<<1)|uint64(1<<2)|uint64(1<<3), p.BCatMask)
	assert.True(t, checkBlocklistsParsed(p.OpenRTB26Hot, &p.OpenRTB26Cold, true))
	assert.True(t, checkBlocklistsParsed(p.OpenRTB26Hot, &p.OpenRTB26Cold, false) == false)
	body2 := []byte(`{
	  "id":"blk-2",
	  "badv":["evil.example"],
	  "imp":[{"id":"1","bidfloor":1.0,"banner":{"w":300,"h":250}}],
	  "site":{"domain":"ok.example"},
	  "device":{"ip":"1.2.3.4","ua":"Mozilla/5.0"}
	}`)
	p2 := ParseOpenRTB26(body2)
	require.True(t, p2.OK)
	assert.False(t, checkBlocklistsParsed(p2.OpenRTB26Hot, &p2.OpenRTB26Cold, true))
}

func TestParseOpenRTB26_exchangeReady(t *testing.T) {
	body := validExchangeBody()
	p := ParseOpenRTB26(body)
	require.True(t, p.ExchangeReady(openrtb.ExchangeConfig{MultiImpMax: 1}))
}

func TestParseOpenRTB26_exchangeReady_rejectsMissingUA(t *testing.T) {
	body := []byte(`{"id":"b1","imp":[{"id":"1","bidfloor":0.5,"banner":{}}],"site":{},"device":{"ip":"1.2.3.4"}}`)
	p := ParseOpenRTB26(body)
	assert.False(t, p.ExchangeReady(openrtb.ExchangeConfig{MultiImpMax: 1}))
}

func TestParseOpenRTB26_multiImp(t *testing.T) {
	body := []byte(`{"id":"multi-1","tmax":300,"imp":[{"id":"a","bidfloor":0.5,"banner":{"w":300,"h":250}},{"id":"b","bidfloor":1.0,"video":{"w":640,"h":480,"maxduration":30}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
	p := ParseOpenRTB26(body)
	require.True(t, p.OK)
	assert.Equal(t, uint8(2), p.ImpCount)
	assert.Equal(t, uint8(2), p.ImpSlots)
	assert.Equal(t, "a", string(p.Imps[0].ImpID[:p.Imps[0].ImpIDLen]))
	assert.Equal(t, "b", string(p.Imps[1].ImpID[:p.Imps[1].ImpIDLen]))
	assert.True(t, p.Imps[0].Flags&impSlotFlagBanner != 0)
	assert.True(t, p.Imps[1].Flags&impSlotFlagVideo != 0)
	assert.Equal(t, int64(500_000), p.Imps[0].BidFloorMicro)
	assert.Equal(t, int64(1_000_000), p.Imps[1].BidFloorMicro)
	assert.True(t, p.ExchangeReady(openrtb.ExchangeConfig{MultiImpMax: 10}))
	assert.False(t, p.ExchangeReady(openrtb.ExchangeConfig{MultiImpMax: 1}))
}

func BenchmarkParseOpenRTB26(b *testing.B) {
	var out OpenRTB26Parsed
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ParseOpenRTB26Into(openrtb26Sample, &out)
	}
}

func BenchmarkParseOpenRTB26Into_connReuse(b *testing.B) {
	var ctx connContext
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ParseOpenRTB26Split(openrtb26Sample, &ctx.openrtbParsed.OpenRTB26Hot, &ctx.openrtbParsed.OpenRTB26Cold)
	}
}

func BenchmarkParseOpenRTB26Split_hotOnly(b *testing.B) {
	var ctx connContext
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ParseOpenRTB26Split(openrtb26Sample, &ctx.openrtbParsed.OpenRTB26Hot, &ctx.openrtbParsed.OpenRTB26Cold)
	}
}

func BenchmarkWriteOpenRTB26BidHTTP(b *testing.B) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "req-1")
	copy(bidID[:], "1")
	copy(impID[:], "1")
	copy(camp[:], "00000000-0000-0000-0000-000000000001")
	adm := []byte(`<html>ad</html>`)
	p := openrtb.BidWire{
		RequestID:  reqID[:len("req-1")],
		BidID:      bidID[:len("1")],
		ImpID:      impID[:len("1")],
		PriceMicro: 1_250_000,
		CurUSD:     true,
		AdM:        adm,
		CampaignID: camp[:36],
		CreativeID: 1,
	}
	var buf [1536]byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = openrtb.WriteBidHTTPResponse(buf[:], p, openrtb.HTTPWriteOpts{})
	}
}
