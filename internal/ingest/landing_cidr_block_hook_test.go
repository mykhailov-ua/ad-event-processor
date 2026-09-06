package ingest

import (
	"net"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func configureClickHookRotationTables(h *AdsPacketHandler) {
	v4 := NewIPv4RotationTable()
	v4.SetMode("shadow")
	v4.SetPolicy(uint64(time.Minute.Nanoseconds()), 1<<20)
	h.ConfigureIPv4Rotation(v4)
	v6 := NewIPv6RotationTable()
	v6.SetMode("shadow")
	v6.SetPolicy(uint64(time.Minute.Nanoseconds()), 1<<20)
	h.ConfigureIPv6Rotation(v6)
}

func clickHookNonMatchingCIDRTable(t *testing.T) *CIDRTable {
	t.Helper()
	return cidrBlockTestTable(t, "10.0.0.0/8")
}

func clickHookBrandStore(t *testing.T, brandID uuid.UUID) *BrandCreativeStore {
	t.Helper()
	store := NewBrandCreativeStore(nil, 0)
	store.SetFixturesForTest(map[uuid.UUID][]BrandCreativeFixture{brandID: {{
		URL:    "https://lander.test/go?cid={click_id}",
		Weight: 100,
	}}})
	return store
}

func cidrBlockHookHandler(t *testing.T, cidrBlockEnabled bool, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	cid := uuid.New()
	brandID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
		c.CIDRBlockEnabled = cidrBlockEnabled
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.CIDRBlockEnabled = false })
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	store := clickHookBrandStore(t, brandID)
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	return h, cid
}

func serveClickFromIP(h *AdsPacketHandler, cid uuid.UUID, ip string) *GnetHarnessConn {
	wire := BuildGnetHTTP("GET", "/click?campaign_id="+cid.String()+"&type=click&gclid=GCLID1", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil)
	conn := NewGnetHarnessConn(wire)
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(ip), Port: 4321})
	h.OnTraffic(conn)
	return conn
}

func cidrBlockTestTable(t *testing.T, cidrs ...string) *CIDRTable {
	t.Helper()
	table, err := BuildCIDRTableFromPrefixes(cidrs...)
	if err != nil {
		t.Fatalf("BuildCIDRTableFromPrefixes: %v", err)
	}
	return table
}

func TestClickRedirect_CIDRBlockMatch_SafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)
	h.ConfigureCIDR(cidrBlockTestTable(t, "54.0.0.0/8"))
	configureClickHookRotationTables(h)

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-View: l1")
	require.Contains(t, resp, "<title>Loading</title>")
	require.Greater(t, len(conn.Written()), len("HTTP/1.1 200 OK\r\n\r\n")+64, "body must not be header-only stub")
	require.Equal(t, 0, filter.calls, "CIDR block match must short-circuit before FilterEngine")
}

func TestClickRedirect_CIDRBlockNoMatch_FallsThroughToFilter(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)
	h.ConfigureCIDR(cidrBlockTestTable(t, "54.0.0.0/8"))
	configureClickHookRotationTables(h)

	conn := serveClickFromIP(h, cid, "8.8.8.8")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls, "non-match must reach FilterEngine")
}

func TestClickRedirect_CIDRBlockCampaignDisabled_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, false, filter)
	h.ConfigureCIDR(cidrBlockTestTable(t, "54.0.0.0/8"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls, "campaign flag off must bypass CIDR block")
}

func TestClickRedirect_CIDRBlockTableNil_FailClosed(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
	require.Equal(t, 0, filter.calls, "CIDR enabled with nil table must safe-view before FilterEngine")
}

func TestClickRedirect_CIDRBlockTableUnpublished_FailClosed(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)
	h.ConfigureCIDR(NewCIDRTable())
	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
	require.Equal(t, 0, filter.calls, "CIDR enabled with unpublished table must safe-view before FilterEngine")
}

// Holdout: revert fail-closed branch and DC-range IP reaches FilterEngine when table nil.
func TestClickRedirect_CIDRBlockTableNil_failOpen_holdout(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	if filter.calls == 1 {
		t.Fatal("holdout: CIDRBlockEnabled with nil table must not fall through to FilterEngine")
	}
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
}

func TestClickRedirect_CIDRBlockMatch_IPv6(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)
	h.ConfigureCIDR(cidrBlockTestTable(t, "2001:db8::/32"))

	conn := serveClickFromIP(h, cid, "2001:db8::dead")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
	require.Equal(t, 0, filter.calls)
}
