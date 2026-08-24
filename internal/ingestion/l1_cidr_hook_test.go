package ingestion

import (
	"net"
	"net/http"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func l1HookHandler(t *testing.T, l1Enabled bool, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.L1CIDRBlockEnabled = l1Enabled
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.L1CIDRBlockEnabled = false })
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
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

func l1TestTable(t *testing.T, cidrs ...string) *CIDRTable {
	t.Helper()
	table, _ := buildTestTable(t, cidrs...)
	return table
}

func TestClickRedirect_L1Match_SafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l1HookHandler(t, true, filter)
	h.ConfigureCIDR(l1TestTable(t, "54.0.0.0/8"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-View: l1")
	require.Contains(t, resp, "<title>Loading</title>")
	require.Greater(t, len(conn.Written()), len("HTTP/1.1 200 OK\r\n\r\n")+64, "body must not be header-only stub")
	require.Equal(t, 0, filter.calls, "L1 match must short-circuit before FilterEngine")
}

func TestClickRedirect_L1NoMatch_FallsThroughToFilter(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l1HookHandler(t, true, filter)
	h.ConfigureCIDR(l1TestTable(t, "54.0.0.0/8"))

	conn := serveClickFromIP(h, cid, "8.8.8.8")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls, "non-match must reach FilterEngine")
}

func TestClickRedirect_L1CampaignDisabled_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l1HookHandler(t, false, filter)
	h.ConfigureCIDR(l1TestTable(t, "54.0.0.0/8"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls, "campaign flag off must bypass L1")
}

func TestClickRedirect_L1TableNil_FailOpen(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l1HookHandler(t, true, filter)

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L1TableUnpublished_FailOpen(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l1HookHandler(t, true, filter)
	h.ConfigureCIDR(NewCIDRTable())
	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L1Match_IPv6(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l1HookHandler(t, true, filter)
	h.ConfigureCIDR(l1TestTable(t, "2001:db8::/32"))

	conn := serveClickFromIP(h, cid, "2001:db8::dead")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
	require.Equal(t, 0, filter.calls)
}
