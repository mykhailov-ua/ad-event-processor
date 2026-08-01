package ingestion

import (
	"net/http"
	"testing"

	"espx/internal/config"
	"espx/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseClickQuery(t *testing.T) {
	t.Parallel()
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	path := []byte("/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&click_id=c1&user_id=u1&sub1=fb&gclid=abc123&ttclid=xyz")
	parsed := &clickQueryParsed{}
	_ = parseClickQuery(path, nil, parsed)
	require.True(t, parsed.ok)
	require.Equal(t, cid, parsed.campaignID)
	require.Equal(t, "click", parsed.eventType)
	require.Equal(t, "c1", parsed.clickID)
	require.Equal(t, "u1", parsed.userID)
	require.Equal(t, "fb", parsed.subs[0])
	require.Contains(t, string(parsed.passthrough), "gclid=abc123")
	require.Contains(t, string(parsed.passthrough), "ttclid=xyz")
}

func TestBuildRedirectLocation_macrosAndPassthrough(t *testing.T) {
	t.Parallel()
	base := []byte("https://offer.test/lp?x=1&cid={click_id}&s={sub1}")
	loc, ok := buildRedirectLocation(nil, base, "click-99", "user-1", [5]string{"fb"}, []byte("gclid=G1"))
	require.True(t, ok)
	require.Equal(t, "https://offer.test/lp?x=1&cid=click-99&s=fb&gclid=G1", string(loc))
}

func TestClickRedirectGnet_302(t *testing.T) {
	t.Parallel()
	cid := uuid.New()
	brandID := uuid.New()
	staticCampaignMu.Lock()
	staticCampaign = &domain.Campaign{
		ID:         cid,
		CustomerID: uuid.Nil,
		BrandID:    &brandID,
		Location:   staticCampaign.Location,
	}
	staticCampaignMu.Unlock()
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil)
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
