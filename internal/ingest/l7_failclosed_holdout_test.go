package ingest

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractClientIP_untrustedRemote_ignoresXFF_holdout(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/track", nil)
	require.NoError(t, err)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.99")

	ip := extractClientIP(req, []string{"1.1.1.1"})
	require.Equal(t, "203.0.113.5", ip, "untrusted remote must not trust attacker XFF")
}

func TestGeoFilter_lookupError_failClosed503_holdout(t *testing.T) {
	campID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = campID
		c.TargetCountries = map[string]struct{}{"US": {}}
	})
	t.Cleanup(resetStaticCampaignBaseline)

	f := NewGeoFilter(errGeoProvider{}, &mockRegistry{})
	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	err := f.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrGeoLookupFailed)

	kind, ok := classifyFilterErr(err)
	require.True(t, ok)
	require.Equal(t, filterRejectInfra, kind)
	require.Equal(t, http.StatusServiceUnavailable, filterRejectSpecs[kind].status)
}

func TestExtractClientIP_trustedProxy_privateXFF_fallsBackToXRealIP_holdout(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/track", nil)
	require.NoError(t, err)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.50, 10.0.0.99")
	req.Header.Set("X-Real-Ip", "203.0.113.44")

	ip := extractClientIP(req, []string{"10.0.0.0/8"})
	require.Equal(t, "203.0.113.44", ip)
}

func TestExtractClientIPGnet_trustedProxy_privateXFF_fallsBackToXRealIP_holdout(t *testing.T) {
	trusted := []string{"10.0.0.0/8"}
	req := parsedHTTPRequest{
		ClientIP: []byte("192.168.1.50, 10.0.0.99"),
		RealIP:   []byte("203.0.113.44"),
	}
	addr := &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 1234}
	ctx := &connContext{}
	conn := NewGnetHarnessConn(nil)
	conn.SetContext(ctx)
	conn.SetRemoteAddr(addr)

	ip := extractClientIPGnet(ctx, &req, conn, trusted)
	require.Equal(t, "203.0.113.44", ip)
}

func TestExtractClientIP_trustedProxy_privateXFFOnly_returnsProxyIP_holdout(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/track", nil)
	require.NoError(t, err)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.50, 10.0.0.99")

	ip := extractClientIP(req, []string{"10.0.0.0/8"})
	require.Equal(t, "10.0.0.1", ip, "private-only XFF without X-Real-IP falls back to proxy IP")
}

func TestExtractClientIPGnet_trustedProxy_privateXFFOnly_returnsProxyIP_holdout(t *testing.T) {
	trusted := []string{"10.0.0.0/8"}
	req := parsedHTTPRequest{ClientIP: []byte("192.168.1.50, 10.0.0.99")}
	addr := &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 1234}
	ctx := &connContext{}
	conn := NewGnetHarnessConn(nil)
	conn.SetContext(ctx)
	conn.SetRemoteAddr(addr)

	ip := extractClientIPGnet(ctx, &req, conn, trusted)
	require.Equal(t, "10.0.0.1", ip)
}

func TestGeoFilter_registryStale_unknownCampaign503_holdout(t *testing.T) {
	reg := NewRegistry(nil)
	reg.ConfigureStaleMode(1 * time.Millisecond)
	time.Sleep(3 * time.Millisecond)
	require.True(t, reg.IsStaleMode())

	f := NewGeoFilter(&MockGeoProvider{}, reg)
	evt := &domain.Event{CampaignID: uuid.New(), IP: "8.8.8.8"}
	err := f.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrRegistryStale)

	kind, ok := classifyFilterErr(err)
	require.True(t, ok)
	require.Equal(t, filterRejectRegistryStale, kind)
	require.Equal(t, http.StatusServiceUnavailable, filterRejectSpecs[kind].status)
}

func TestGeoFilter_registryFresh_unknownCampaign404_not503_holdout(t *testing.T) {
	reg := NewRegistry(nil)
	reg.MarkPubSubOK()
	require.False(t, reg.IsStaleMode())

	f := NewGeoFilter(&MockGeoProvider{}, reg)
	evt := &domain.Event{CampaignID: uuid.New(), IP: "8.8.8.8"}
	err := f.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrCampaignNotFound)

	kind, ok := classifyFilterErr(err)
	require.True(t, ok)
	require.Equal(t, filterRejectCampaignNotFound, kind)
	require.Equal(t, http.StatusNotFound, filterRejectSpecs[kind].status)
}

func TestGeoFilter_lookupError_failOpen_passes_holdout(t *testing.T) {
	campID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = campID
		c.TargetCountries = map[string]struct{}{"US": {}}
	})
	t.Cleanup(resetStaticCampaignBaseline)

	f := NewGeoFilter(errGeoProvider{}, &mockRegistry{})
	f.SetGeoFailClosed(false)
	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	err := f.Check(context.Background(), evt)
	require.NoError(t, err, "GEO_FAIL_CLOSED=0 keeps fail-open on geo lookup error")
}

func TestProcessTrack_registryStale503_holdout(t *testing.T) {
	reg := NewRegistry(nil)
	reg.ConfigureStaleMode(1 * time.Millisecond)
	time.Sleep(3 * time.Millisecond)

	geo := &MockGeoProvider{}
	engine := NewFilterEngine(0, NewGeoFilter(geo, reg))
	engine.SetRegistry(reg)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.CampaignID = uuid.New()
	evt.Type = "click"

	out := processTrack(context.Background(), newTrackProcessor(engine, reg, nil), evt, nil)
	require.Equal(t, trackStatusRejected, out.Status)
	require.Equal(t, filterRejectRegistryStale, out.RejectKind)
	require.Equal(t, http.StatusServiceUnavailable, filterRejectSpecs[out.RejectKind].status)
}

func TestPostTrackGnetJSON_registryStale503_holdout(t *testing.T) {
	reg := NewRegistry(nil)
	reg.ConfigureStaleMode(1 * time.Millisecond)
	time.Sleep(3 * time.Millisecond)
	require.True(t, reg.IsStaleMode())

	cfg := &config.Config{
		MaxRequestBodySize: 1024 * 1024,
		FilterTimeoutMs:    50,
	}
	geo := &MockGeoProvider{}
	h := NewAdsPacketHandler(
		cfg,
		reg,
		NewFilterEngine(50*time.Millisecond, NewGeoFilter(geo, reg)),
		nil,
		nil,
		NewJumpHashSharder(1),
		"fraud-stream",
		nil,
	)
	body := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
	status, respBody := PostTrackGnetJSON(h, body)
	require.Equal(t, http.StatusServiceUnavailable, status)
	assert.Contains(t, string(respBody), "registry_stale")
}

func TestPostTrackGnetJSON_unknownCampaign404_whenRegistryFresh_holdout(t *testing.T) {
	reg := NewRegistry(nil)
	reg.MarkPubSubOK()

	cfg := &config.Config{
		MaxRequestBodySize: 1024 * 1024,
		FilterTimeoutMs:    50,
	}
	h := NewAdsPacketHandler(
		cfg,
		reg,
		NewFilterEngine(50*time.Millisecond, NewGeoFilter(&MockGeoProvider{}, reg)),
		nil,
		nil,
		NewJumpHashSharder(1),
		"fraud-stream",
		nil,
	)
	body := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
	status, _ := PostTrackGnetJSON(h, body)
	require.Equal(t, http.StatusNotFound, status)
}

func TestStreamProducerAdmission_default85Pct_rejectsAtPressure_holdout(t *testing.T) {
	p := NewStreamProducerQueueForTest(100, 86)

	cfg := &config.Config{StreamProducerAdmissionPct: 85}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	_, kind, acquired := tryAcquireStreamAdmission(cfg, sharder, producers, nil, campaignID, false)
	require.False(t, acquired)
	require.Equal(t, filterRejectProducerOverload, kind)
	require.Equal(t, http.StatusServiceUnavailable, filterRejectSpecs[kind].status)
}

func TestStreamProducerAdmission_tunablePct50_rejectsAtHalfFill_holdout(t *testing.T) {
	p := NewStreamProducerQueueForTest(100, 50)

	cfg := &config.Config{StreamProducerAdmissionPct: 50}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	_, kind, acquired := tryAcquireStreamAdmission(cfg, sharder, producers, nil, campaignID, false)
	require.False(t, acquired)
	require.Equal(t, filterRejectProducerOverload, kind)
}
