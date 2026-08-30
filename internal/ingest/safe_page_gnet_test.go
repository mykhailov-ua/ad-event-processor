package ingest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestClickRedirectGnet_forceSafeInPlace(t *testing.T) {
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &countingFilter{})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	path := "/click?campaign_id=" + cid.String() + "&type=click&gclid=GCLID1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":                      "keep-alive",
		"Content-Length":                  "0",
		"User-Agent":                      "Mozilla/5.0",
		"X-ad-event-processor-Force-Safe": "1",
	}, nil))

	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-Page: 1")
}

func TestClickRedirectGnet_fraudRedirect(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	configureMockRegistryCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
		c.SilentRejectEnabled = true
	})
	t.Cleanup(resetStaticCampaignBaseline)

	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{URL: "https://lander.test/go", Weight: 100}}})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &fraudRejectFilter{})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)

	path := "/click?campaign_id=" + cid.String() + "&type=click&gclid=GCLID1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))

	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "Location: https://safe.example/white")
}

func TestClickRedirectGnet_fraudSilentReject_noSafePage_not204(t *testing.T) {
	cid := uuid.New()
	configureMockRegistryCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.SilentRejectEnabled = true
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	engine := NewFilterEngine(0, &fraudRejectFilter{})
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, engine, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)

	path := "/click?campaign_id=" + cid.String() + "&type=click"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))

	status := ParseGnetHTTPStatus(conn.Written())
	require.NotEqual(t, http.StatusNoContent, status)
	require.Contains(t, []int{http.StatusFound, http.StatusOK}, status)
}

type fraudRejectFilter struct{}

func (f *fraudRejectFilter) Check(_ context.Context, _ *domain.Event) error {
	return ErrFraudDetected
}

func TestSafePageStub_embedsHydrator(t *testing.T) {
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/embed"
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	path := "/safe_page_stub?campaign_id=" + cid.String()
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))

	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "safe.example/embed")
	require.Contains(t, resp, "/track/verify")
	require.Contains(t, resp, string(safePageHydratorJS[:40]))
}

func TestTrackVerify_success(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/"
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{URL: "https://money.example/lp", Weight: 100}}})

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
	require.NoError(t, err)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	wire := BuildGnetHTTP("POST", safePageVerifyPath, map[string]string{
		"Content-Type": "application/json",
		"Connection":   "keep-alive",
	}, body)
	_, conn := ServeGnetHarness(h, wire)
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, `"success":true`)
	require.Contains(t, resp, "money.example")
}
