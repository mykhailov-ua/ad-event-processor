package ingestion

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func attestationHookHandler(t *testing.T, filter EventFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
		c.AttestationEnabled = true
		c.AttestationMode = domain.AttestationModeStrict
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.AttestationEnabled = false
			c.AttestationMode = domain.AttestationModeOff
			c.SafePageEnabled = false
			c.SafePageURL = ""
		})
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureAttestation([][]byte{secret})
	return h, cid
}

func serveClickWithCookie(h *AdsPacketHandler, cid uuid.UUID, ip string, cookie string) *GnetHarnessConn {
	headers := map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}
	if cookie != "" {
		headers["Cookie"] = cookie
	}
	wire := BuildGnetHTTP("GET", "/click?campaign_id="+cid.String()+"&type=click", headers, nil)
	conn := NewGnetHarnessConn(wire)
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(ip), Port: 4321})
	h.OnTraffic(conn)
	return conn
}

func TestHTTP1CookieHeaderParsed(t *testing.T) {
	wire := BuildGnetHTTP("GET", "/click?campaign_id=x", map[string]string{
		"Cookie":         "Attestation-Token=abc123",
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil)
	_, req, err := parseHTTP1(wire, 1<<20, nil)
	require.NoError(t, err)
	require.Equal(t, []byte("Attestation-Token=abc123"), req.Cookie)
}

func TestClickRedirect_AttestationStrictMissing_ServesStub(t *testing.T) {
	filter := &countingFilter{}
	h, cid := attestationHookHandler(t, filter)
	conn := serveClickWithCookie(h, cid, "203.0.113.5", "")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	body := string(conn.Written())
	require.Contains(t, body, "<script>")
	require.Equal(t, 0, filter.calls, "strict attestation must short-circuit before FilterEngine")
}

func TestClickRedirect_AttestationValidCookie_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := attestationHookHandler(t, filter)
	ip := "203.0.113.5"
	now := time.Now().Unix()
	token, err := MintAttestationToken(h.attestationKeys[0].secret, cid, ip, 300, now)
	require.NoError(t, err)
	require.True(t, h.verifyAttestationCookie([]byte("Attestation-Token="+token), cid, ip, now+1))
	conn := serveClickWithCookie(h, cid, ip, "Attestation-Token="+token)
	written := string(conn.Written())
	require.NotContains(t, written, "<iframe src=", "valid cookie must not serve safe-page stub")
}

func TestClickRedirect_AttestationDisabled_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := attestationHookHandler(t, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.AttestationEnabled = false
		c.AttestationMode = domain.AttestationModeOff
	})
	_ = serveClickWithCookie(h, cid, "203.0.113.5", "")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_AttestationLightMissing_L2AndSafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := attestationHookHandler(t, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.AttestationMode = domain.AttestationModeLight
		c.AttestationEnabled = false
	})
	conn := serveClickWithCookie(h, cid, "203.0.113.5", "")
	written := string(conn.Written())
	require.Equal(t, 1, filter.calls, "light attestation must run FilterEngine")
	require.Contains(t, written, "X-ad-event-processor-Safe-Page: 1", "light mode serves safe view header")
}

type missingImpSignalFilter struct{}

func (f missingImpSignalFilter) Check(_ context.Context, evt *domain.Event) error {
	addFraudSignal(evt, FraudReasonMissingImpTS)
	return nil
}

func TestClickRedirect_AttestationCookieMissingImp_ForceSafe(t *testing.T) {
	h, cid := attestationHookHandler(t, missingImpSignalFilter{})
	lockStaticCampaign(func(c *domain.Campaign) {
		c.AttestationMode = domain.AttestationModeLight
		c.AttestationEnabled = false
	})
	ip := "203.0.113.7"
	now := time.Now().Unix()
	token, err := MintAttestationToken(h.attestationKeys[0].secret, cid, ip, 300, now)
	require.NoError(t, err)
	conn := serveClickWithCookie(h, cid, ip, "Attestation-Token="+token)
	written := string(conn.Written())
	require.Contains(t, written, "X-ad-event-processor-Safe-Page: 1", "attestation + missing imp forces safe view")
}

func TestClickRedirect_AttestationOffMissingImp_NoForceSafe(t *testing.T) {
	h, cid := attestationHookHandler(t, missingImpSignalFilter{})
	lockStaticCampaign(func(c *domain.Campaign) {
		c.AttestationMode = domain.AttestationModeOff
		c.AttestationEnabled = false
		c.SafePageEnabled = false
	})
	conn := serveClickWithCookie(h, cid, "203.0.113.8", "")
	written := string(conn.Written())
	require.NotContains(t, written, "X-ad-event-processor-Safe-Page: 1")
}
