package ingestion

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func attestationHookHandler(t *testing.T, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.SafePageEnabled = true
		c.SafePageURL = "https://safe.example/white"
		c.AttestationEnabled = true
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.AttestationEnabled = false
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

func TestClickRedirect_AttestationMissing_ServesStub(t *testing.T) {
	filter := &countingFilter{}
	h, cid := attestationHookHandler(t, filter)
	conn := serveClickWithCookie(h, cid, "203.0.113.5", "")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	body := string(conn.Written())
	require.Contains(t, body, "<script>")
	require.Contains(t, body, "BidShard")
	require.Equal(t, 0, filter.calls, "attestation stub must short-circuit before FilterEngine")
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
	lockStaticCampaign(func(c *domain.Campaign) { c.AttestationEnabled = false })
	_ = serveClickWithCookie(h, cid, "203.0.113.5", "")
	require.Equal(t, 1, filter.calls)
}
