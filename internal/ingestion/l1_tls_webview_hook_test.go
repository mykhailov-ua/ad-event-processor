package ingestion

import (
	"net"
	"net/http"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func serveClickWithJA3UA(h *AdsPacketHandler, cid uuid.UUID, ip, ja3, ua string) *GnetHarnessConn {
	wire := BuildGnetHTTP("GET", "/click?campaign_id="+cid.String()+"&type=click", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     ua,
		"X-TLS-JA3":      ja3,
	}, nil)
	conn := NewGnetHarnessConn(wire)
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(ip), Port: 4321})
	h.OnTraffic(conn)
	return conn
}

func TestClickRedirect_SocialInAppWebView_TLSRelax_NoSafeView(t *testing.T) {
	ja3 := "771,4865-4866,0-23,29-23-24,0"
	inAppUA := "Mozilla/5.0 [FBAN/FB4A;FBAV/128.0.0.0;]"
	filter := &countingFilter{}
	h, cid := tlsHookHandler(t, true, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.SocialInAppEnabled = true
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.SocialInAppEnabled = false })
	})
	h.ConfigureTLSFingerprint(buildTestTLSFingerprintTable("ja3:" + ja3))

	conn := serveClickWithJA3UA(h, cid, "8.8.8.8", ja3, inAppUA)
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: tls")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_SocialInAppWebView_TLSRelax_AttestationLightStillL2(t *testing.T) {
	ja3 := "771,4865-4866,0-23,29-23-24,0"
	inAppUA := "Mozilla/5.0 [FBAN/FB4A;FBAV/128.0.0.0;]"
	filter := &countingFilter{}
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.TLSFingerprintBlockEnabled = true
		c.SocialInAppEnabled = true
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
		c.AttestationMode = domain.AttestationModeLight
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.SocialInAppEnabled = false
			c.SafePageEnabled = false
			c.SafePageURL = ""
			c.AttestationMode = domain.AttestationModeOff
		})
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureTLSFingerprint(buildTestTLSFingerprintTable("ja3:" + ja3))
	h.ConfigureAttestation([][]byte{secret})

	conn := serveClickWithJA3UA(h, cid, "8.8.8.8", ja3, inAppUA)
	written := string(conn.Written())
	require.NotContains(t, written, "X-ad-event-processor-Safe-View: tls")
	require.Equal(t, 1, filter.calls, "TLS relax must not skip FilterEngine")
	require.Contains(t, written, "X-ad-event-processor-Safe-Page: 1", "L2 attestation must still apply")
}

func TestClickRedirect_SocialInApp_BotUA_TLSStillSafeView(t *testing.T) {
	ja3 := "771,4865-4866,0-23,29-23-24,0"
	filter := &countingFilter{}
	h, cid := tlsHookHandler(t, true, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.SocialInAppEnabled = true
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.SocialInAppEnabled = false })
	})
	h.ConfigureTLSFingerprint(buildTestTLSFingerprintTable("ja3:" + ja3))

	conn := serveClickWithJA3UA(h, cid, "8.8.8.8", ja3, "curl/8.0")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: tls")
	require.Equal(t, 0, filter.calls)
}
