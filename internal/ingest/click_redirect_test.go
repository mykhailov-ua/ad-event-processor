package ingest

import (
	"bytes"
	"net/http"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseClickQuery(t *testing.T) {
	t.Parallel()
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&click_id=c1&user_id=u1&sub1=fb&gclid=abc123&ttclid=xyz")
	parsed := &clickQueryParsed{}
	_ = parseClickQuery(path, nil, parsed)
	require.True(t, parsed.OK)
	require.Equal(t, cid, parsed.CampaignID)
	require.Equal(t, "click", parsed.EventType)
	require.Equal(t, "c1", parsed.ClickID)
	require.Equal(t, "u1", parsed.UserID)
	require.Equal(t, "fb", parsed.Subs[0])
	require.Equal(t, "abc123", parsed.GCLID)
	require.Equal(t, "xyz", parsed.TTCLID)
}

func TestParseClickQuery_ingressCostParam(t *testing.T) {
	t.Parallel()
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&cost=0.05&cpc=0.02&bid=0.01")
	parsed := &clickQueryParsed{}
	_ = parseClickQuery(path, nil, parsed)
	require.True(t, parsed.OK)
	require.Equal(t, []byte("0.05"), parsed.IngressCost)
	require.Equal(t, []byte("0.02"), parsed.IngressCPC)
	require.Equal(t, []byte("0.01"), parsed.IngressBid)
}

func TestParseClickQuery_sub30(t *testing.T) {
	t.Parallel()
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click" +
		"&sub1=a&sub10=j&sub20=t&sub30=end")
	parsed := &clickQueryParsed{}
	_ = parseClickQuery(path, nil, parsed)
	require.True(t, parsed.OK)
	require.Equal(t, "a", parsed.Subs[0])
	require.Equal(t, "j", parsed.Subs[9])
	require.Equal(t, "t", parsed.Subs[19])
	require.Equal(t, "end", parsed.Subs[29])
}

func TestParseClickQuery_smokeFlag(t *testing.T) {
	t.Parallel()
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&smoke=1")
	parsed := &clickQueryParsed{}
	_ = parseClickQuery(path, nil, parsed)
	require.True(t, parsed.OK)
	require.True(t, parsed.Smoke)
}

func TestBuildRedirectLocation_sub30Macro(t *testing.T) {
	t.Parallel()
	var subs SubIDSlots
	subs[29] = "slot30"
	base := []byte("https://offer.test/lp?s={sub30}")
	loc, ok := buildRedirectLocation(nil, base, "c1", "u1", subs, nil)
	require.True(t, ok)
	require.Equal(t, "https://offer.test/lp?s=slot30", string(loc))
}

func TestRedirectHdrSuffix_referrerPolicy(t *testing.T) {
	t.Parallel()
	require.Contains(t, redirectHdrSuffix, "Referrer-Policy: no-referrer")
}

func TestBuildRedirectLocation_macrosAndPassthrough(t *testing.T) {
	t.Parallel()
	base := []byte("https://offer.test/lp?x=1&cid={click_id}&s={sub1}")
	loc, ok := buildRedirectLocation(nil, base, "click-99", "user-1", SubIDSlots{"fb"}, []byte("gclid=G1"))
	require.True(t, ok)
	require.Equal(t, "https://offer.test/lp?x=1&cid=click-99&s=fb&gclid=G1", string(loc))
}

func TestBuildRedirectLocation_encodesMacroValues(t *testing.T) {
	t.Parallel()
	base := []byte("https://offer.test/lp?s={sub1}&u={user_id}")
	loc, ok := buildRedirectLocation(nil, base, "c1", "a&b=c", SubIDSlots{"x y"}, nil)
	require.True(t, ok)
	require.Equal(t, "https://offer.test/lp?s=x%20y&u=a%26b%3Dc", string(loc))
}

func requireGnetDmrResponse(t *testing.T, written []byte, landingSnippet string) {
	t.Helper()
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(written))
	resp := string(written)
	require.Contains(t, resp, "Content-Type: text/html; charset=utf-8")
	require.Contains(t, resp, `http-equiv="refresh"`)
	require.Contains(t, resp, `window.location.replace(`)
	idx := bytes.Index(written, []byte("\r\n\r\n"))
	require.True(t, idx > 0, "missing HTTP body")
	body := written[idx+4:]
	require.Greater(t, len(body), 80, "DMR body must include landing URL, not header-only")
	require.Contains(t, string(body), landingSnippet)
	require.NotContains(t, string(body), `"></script><script>`)
}

func setupClickRedirectHarness(t *testing.T, mut func(*domain.Campaign)) (*AdsPacketHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	cid := uuid.New()
	brandID := uuid.New()
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:         cid,
			CustomerID: uuid.Nil,
			BrandID:    &brandID,
			Location:   (*campPtr).Location,
		}
		if mut != nil {
			mut(*campPtr)
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			brandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://lander.test/go?cid={click_id}",
				Weight: 100,
			}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	return h, cid, brandID
}

func TestClickRedirectGnet_DMR_queryFlag(t *testing.T) {
	h, cid, _ := setupClickRedirectHarness(t, nil)
	path := "/click?campaign_id=" + cid.String() + "&type=click&user_id=u1&dmr=1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	requireGnetDmrResponse(t, conn.Written(), "lander.test/go")
}

func TestClickRedirectGnet_DMR_campaignEnabled(t *testing.T) {
	h, cid, _ := setupClickRedirectHarness(t, func(c *domain.Campaign) {
		c.DmrEnabled = true
	})
	path := "/click?campaign_id=" + cid.String() + "&type=click&user_id=u1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	requireGnetDmrResponse(t, conn.Written(), "lander.test/go")
}

func TestClickRedirectGnet_302(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:         cid,
			CustomerID: uuid.Nil,
			BrandID:    &brandID,
			Location:   (*campPtr).Location,
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			brandID: brandCreativeEntriesReady([]brandCreativeEntry{{URL: "https://lander.test/go?cid={click_id}", Weight: 100}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	path := "/click?campaign_id=" + cid.String() + "&type=click&user_id=u1&gclid=GCLID1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "Location: https://lander.test/go?cid=")
	require.Contains(t, resp, "Referrer-Policy: no-referrer")
	require.Contains(t, resp, "gclid=GCLID1")
}

func TestClickRedirectGnet_invalidCampaign(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", "/click?campaign_id=not-a-uuid", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))
	require.Equal(t, http.StatusBadRequest, ParseGnetHTTPStatus(conn.Written()))
}
