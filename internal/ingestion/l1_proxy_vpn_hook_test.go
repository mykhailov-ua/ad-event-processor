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

func l15HookHandler(t *testing.T, l15Enabled bool, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.L15ProxyVPNBlockEnabled = l15Enabled
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.L15ProxyVPNBlockEnabled = false })
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	return h, cid
}

func buildTestProxyVPNTable(t *testing.T, lines ...string) *ProxyVPNTable {
	t.Helper()
	var b proxyVPNBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for _, line := range lines {
		prefix, connType, asn, ok := parseProxyVPNFeedLine(line)
		if !ok {
			t.Fatalf("bad proxy vpn feed line: %q", line)
		}
		b.addPrefix(prefix, connType, asn, &root4, &root6)
	}
	table := NewProxyVPNTable()
	table.Publish(b.snapshot(root4, root6, 1))
	return table
}

func TestClickRedirect_L15Match_SafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-BidShard-Safe-View: l15")
	require.Greater(t, len(conn.Written()), len("HTTP/1.1 200 OK\r\n\r\n")+64)
	require.Equal(t, 0, filter.calls)
}

func TestClickRedirect_L15NoMatch_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "8.8.8.8")
	require.NotContains(t, string(conn.Written()), "X-BidShard-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L15ISPOnly_NoSafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 12345 isp"))

	conn := serveClickFromIP(h, cid, "54.1.2.3")
	require.NotContains(t, string(conn.Written()), "X-BidShard-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L15CampaignDisabled_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, false, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-BidShard-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L15TableNil_FailOpen(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-BidShard-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_L15RunsAfterL1Miss(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	h.ConfigureCIDR(l1TestTable(t, "10.0.0.0/8"))
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Contains(t, string(conn.Written()), "X-BidShard-Safe-View: l15")
	require.Equal(t, 0, filter.calls)
}

func TestClickRedirect_L15RunsAfterL1Hit(t *testing.T) {
	filter := &countingFilter{}
	h, cid := l15HookHandler(t, true, filter)
	h.ConfigureCIDR(l1TestTable(t, "54.0.0.0/8"))
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Contains(t, string(conn.Written()), "X-BidShard-Safe-View: l1")
	require.Equal(t, 0, filter.calls)
}

func TestL15ProxyVPNShouldSafeView_remoteAddr(t *testing.T) {
	h, cid := l15HookHandler(t, true, &countingFilter{})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "203.0.113.0/24 9009 hosting"))
	match, _ := h.l15ProxyVPNShouldSafeView("203.0.113.44", cid)
	require.True(t, match)
}

func TestL15ProxyVPNShouldSafeView_loopbackNoMatch(t *testing.T) {
	h, cid := l15HookHandler(t, true, &countingFilter{})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "203.0.113.0/24 9009 hosting"))
	match, _ := h.l15ProxyVPNShouldSafeView(net.IPv4(127, 0, 0, 1).String(), cid)
	require.False(t, match)
}
