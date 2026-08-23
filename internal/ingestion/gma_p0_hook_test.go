package ingestion

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func tlsHookHandler(t *testing.T, tlsEnabled bool, filter EventFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.TLSFingerprintBlockEnabled = tlsEnabled
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.TLSFingerprintBlockEnabled = false })
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	return h, cid
}

func serveClickWithJA3(h *AdsPacketHandler, cid uuid.UUID, ip, ja3 string) *GnetHarnessConn {
	wire := BuildGnetHTTP("GET", "/click?campaign_id="+cid.String()+"&type=click", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
		"X-TLS-JA3":      ja3,
	}, nil)
	conn := NewGnetHarnessConn(wire)
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(ip), Port: 4321})
	h.OnTraffic(conn)
	return conn
}

func buildTestTLSFingerprintTable(ja3Line string) *TLSFingerprintTable {
	ja3, ja4 := parseTLSFingerprintFeed([]byte(ja3Line))
	table := NewTLSFingerprintTable()
	table.Publish(buildTLSFingerprintSnapshot(ja3, ja4, nil, nil, 1))
	return table
}

func TestHTTP1Assign_TLSJA3Direct(t *testing.T) {
	var req parsedHTTPRequest
	var flags uint8
	var cl int
	key := []byte("X-TLS-JA3")
	val := []byte("771,4865-4866")
	require.NoError(t, http1AssignHeader(&req, key, val, &flags, &cl))
	require.Equal(t, "771,4865-4866", string(req.TLSJA3))
}

func TestHTTP1Parse_TLSJA3Header(t *testing.T) {
	raw := []byte("GET /click HTTP/1.1\r\nX-TLS-JA3: 771,4865-4866\r\nUser-Agent: test\r\nContent-Length: 0\r\n\r\n")
	_, req, err := parseHTTP1(raw, 1<<20, nil)
	require.NoError(t, err)
	require.Equal(t, "771,4865-4866", string(req.TLSJA3))
}

func TestClickRedirect_TLSFingerprintMatch_SafeView(t *testing.T) {
	ja3 := "771,4865-4866,0-23,29-23-24,0"
	filter := &countingFilter{}
	h, cid := tlsHookHandler(t, true, filter)
	h.ConfigureTLSFingerprint(buildTestTLSFingerprintTable("ja3:" + ja3))

	conn := serveClickWithJA3(h, cid, "8.8.8.8", ja3)
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-View: tls")
	require.Contains(t, resp, "<title>Loading</title>")
	require.Greater(t, len(conn.Written()), len("HTTP/1.1 200 OK\r\n\r\n")+64)
	require.Equal(t, 0, filter.calls)
}

func TestClickRedirect_TLSFingerprintNoMatch_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := tlsHookHandler(t, true, filter)
	h.ConfigureTLSFingerprint(buildTestTLSFingerprintTable("ja3:771,4865"))

	conn := serveClickWithJA3(h, cid, "8.8.8.8", "other-fingerprint")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: tls")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L15MobileOnly_RejectsHosting(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ConnTypePolicy = domain.ConnTypeMobileOnly
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.ConnTypePolicy = domain.ConnTypeBlockVPNHosting })
	})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "203.0.113.0/24 9009 hosting"))

	conn := serveClickFromIP(h, cid, "203.0.113.50")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 0, filter.calls)
}

func TestClickRedirect_SignedOfferLink_AttestationCapsTTL(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	staticCampaignMu.Lock()
	staticCampaign = &domain.Campaign{
		ID:                 cid,
		CustomerID:         uuid.Nil,
		BrandID:            &brandID,
		Location:           time.UTC,
		LinkSigningEnabled: true,
		LinkSigningTTLSec:  900,
		AttestationMode:    domain.AttestationModeLight,
	}
	staticCampaignMu.Unlock()
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			brandID: brandCreativeEntriesReady([]brandCreativeEntry{{URL: "https://offer.test/lp", Weight: 100}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	h.ConfigureLinkSigning([]byte("test-link-secret"))

	now := time.Now().Unix()
	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=clk-1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "expires=")
	idx := strings.Index(resp, "expires=")
	require.Greater(t, idx, 0)
	end := idx + len("expires=")
	for end < len(resp) && resp[end] >= '0' && resp[end] <= '9' {
		end++
	}
	expires, err := strconv.ParseInt(resp[idx+len("expires="):end], 10, 64)
	require.NoError(t, err)
	require.LessOrEqual(t, expires-now, int64(linkSigningTTLAttestationCap)+2)
}

func TestClickRedirect_SignedOfferLink_AppendsSig(t *testing.T) {
	cid := uuid.New()
	brandID := uuid.New()
	staticCampaignMu.Lock()
	staticCampaign = &domain.Campaign{
		ID:                 cid,
		CustomerID:         uuid.Nil,
		BrandID:            &brandID,
		Location:           time.UTC,
		LinkSigningEnabled: true,
		LinkSigningTTLSec:  900,
	}
	staticCampaignMu.Unlock()
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			brandID: brandCreativeEntriesReady([]brandCreativeEntry{{URL: "https://offer.test/lp", Weight: 100}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	h.ConfigureLinkSigning([]byte("test-link-secret"))

	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=clk-1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "_sig=")
	require.Contains(t, resp, "expires=")
}

func TestClickRedirect_SignedOfferLink_InvalidSig403(t *testing.T) {
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, &countingFilter{}), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureLinkSigning([]byte("test-link-secret"))

	expires := time.Now().Add(10 * time.Minute).Unix()
	path := "/click?campaign_id=" + cid.String() + "&type=click&click_id=clk-1&expires=" +
		itoa64(expires) + "&_sig=00000000000000000000000000000000"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil))
	require.Equal(t, http.StatusForbidden, ParseGnetHTTPStatus(conn.Written()))
}

func itoa64(v int64) string {
	return string(appendInt64(nil, v))
}
