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

func proxyVPNBlockHookHandler(t *testing.T, proxyVPNBlockEnabled bool, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.ProxyVPNBlockEnabled = proxyVPNBlockEnabled
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.ProxyVPNBlockEnabled = false })
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

func TestClickRedirect_ProxyVPNBlockMatch_SafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-View: l15")
	require.Greater(t, len(conn.Written()), len("HTTP/1.1 200 OK\r\n\r\n")+64)
	require.Equal(t, 0, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockNoMatch_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "8.8.8.8")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockISPOnly_NoSafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 12345 isp"))

	conn := serveClickFromIP(h, cid, "54.1.2.3")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockCampaignDisabled_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, false, filter)
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockTableNil_FailOpen(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockAfterCIDRBlockMiss(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	h.ConfigureCIDR(cidrBlockTestTable(t, "10.0.0.0/8"))
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 0, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockAfterCIDRBlockHit(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	h.ConfigureCIDR(cidrBlockTestTable(t, "54.0.0.0/8"))
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "54.0.0.0/8 16509 vpn"))

	conn := serveClickFromIP(h, cid, "54.230.17.9")
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
	require.Equal(t, 0, filter.calls)
}

func TestProxyVPNBlockShouldSafeView_remoteAddr(t *testing.T) {
	h, cid := proxyVPNBlockHookHandler(t, true, &countingFilter{})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "203.0.113.0/24 9009 hosting"))
	match, _ := h.proxyVPNBlockShouldSafeView("203.0.113.44", cid)
	require.True(t, match)
}

func TestProxyVPNBlockShouldSafeView_loopbackNoMatch(t *testing.T) {
	h, cid := proxyVPNBlockHookHandler(t, true, &countingFilter{})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "203.0.113.0/24 9009 hosting"))
	match, _ := h.proxyVPNBlockShouldSafeView(net.IPv4(127, 0, 0, 1).String(), cid)
	require.False(t, match)
}

func TestConnTypePolicy_mobileOnly_allowsMobileBlocksHosting(t *testing.T) {
	require.False(t, connTypePolicyBlocks(domain.ConnTypeMobileOnly, true, ProxyVPNConnMobile))
	require.True(t, connTypePolicyBlocks(domain.ConnTypeMobileOnly, true, ProxyVPNConnHosting))
	require.True(t, connTypePolicyBlocks(domain.ConnTypeMobileOnly, false, 0))
}

func TestConnTypePolicy_residentialOnly_blocksMobile_holdout(t *testing.T) {
	require.True(t, connTypePolicyBlocks(domain.ConnTypeResidentialOnly, true, ProxyVPNConnMobile|ProxyVPNConnISP))
	require.False(t, connTypePolicyBlocks(domain.ConnTypeResidentialOnly, true, ProxyVPNConnISP))
}

func TestClickRedirect_ProxyVPNBlockMobileOnly_AllowsMobileCarrier(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ConnTypePolicy = domain.SocialInAppConnTypePolicy
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.ConnTypePolicy = domain.ConnTypeBlockVPNHosting
		})
	})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "100.64.0.0/24 310410 mobile"))

	conn := serveClickFromIP(h, cid, "100.64.0.1")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_ProxyVPNBlockResidentialOnly_BlocksMobileCarrier(t *testing.T) {
	filter := &countingFilter{}
	h, cid := proxyVPNBlockHookHandler(t, true, filter)
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ConnTypePolicy = domain.ConnTypeResidentialOnly
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.ConnTypePolicy = domain.ConnTypeBlockVPNHosting
		})
	})
	h.ConfigureProxyVPN(buildTestProxyVPNTable(t, "100.64.0.0/24 310410 mobile"))

	conn := serveClickFromIP(h, cid, "100.64.0.1")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l15")
	require.Equal(t, 0, filter.calls)
}
