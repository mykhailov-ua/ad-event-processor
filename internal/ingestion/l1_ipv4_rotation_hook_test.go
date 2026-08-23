package ingestion

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func ipv4RotationHandler(t *testing.T, l1Enabled bool, mode string, threshold uint32, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	h, cid := l1HookHandler(t, l1Enabled, filter)
	table := NewIPv4RotationTable()
	table.SetMode(mode)
	table.SetPolicy(uint64(time.Minute.Nanoseconds()), threshold)
	h.ConfigureIPv4Rotation(table)
	return h, cid
}

func rotatedIPv4Host(n int) string {
	return fmt.Sprintf("203.0.113.%d", n)
}

func serveClickFromIPUser(h *AdsPacketHandler, cid uuid.UUID, ip, userID string) *GnetHarnessConn {
	wire := BuildGnetHTTP("GET", "/click?campaign_id="+cid.String()+"&type=click&user_id="+userID+"&gclid=GCLID1", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
		"User-Agent":     "Mozilla/5.0",
	}, nil)
	conn := NewGnetHarnessConn(wire)
	conn.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(ip), Port: 4321})
	h.OnTraffic(conn)
	return conn
}

func TestClickRedirect_IPv4Rotation_Live_SafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "live", 4, filter)

	for i := 1; i <= 3; i++ {
		conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(i), "user-a")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	}
	conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(4), "user-a")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-View: l1v4")
	require.Equal(t, 3, filter.calls, "live block must short-circuit before FilterEngine")
}

func TestClickRedirect_IPv4Rotation_Shadow_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "shadow", 3, filter)

	for i := 1; i <= 4; i++ {
		conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(i), "user-a")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
		require.Equal(t, i, filter.calls, "shadow must reach FilterEngine on every click")
	}
}

func TestClickRedirect_IPv4Rotation_UserScoped(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "live", 4, filter)

	for i := 1; i <= 3; i++ {
		conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(i), "user-a")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	}
	conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(4), "user-b")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 4, filter.calls)
}

func TestClickRedirect_IPv4Rotation_Shadow_L2Signal(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "shadow", 2, filter)

	serveClickFromIPUser(h, cid, rotatedIPv4Host(1), "user-a")
	require.Equal(t, 1, filter.calls)

	conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(2), "user-a")
	require.Equal(t, 2, filter.calls)
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
}

func TestIPv4Rotation_L2SignalThroughFilterEngine(t *testing.T) {
	engine := NewFilterEngine(0, &countingFilter{})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	attachFraudAccumulator(evt)
	addFraudSignal(evt, FraudReasonIPv4Rotation)
	require.NoError(t, engine.Check(context.Background(), evt))
	require.Contains(t, evt.FraudReason, FraudReasonCodeIPv4Rotation)
	require.True(t, evt.ShadowEvent)
}

func TestClickRedirect_IPv4Rotation_CampaignDisabled(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, false, "live", 2, filter)

	for i := 1; i <= 3; i++ {
		conn := serveClickFromIPUser(h, cid, rotatedIPv4Host(i), "user-a")
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	}
	require.Equal(t, 3, filter.calls)
}

func TestClickRedirect_IPv4Rotation_IPv6Skipped(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv4RotationHandler(t, true, "live", 2, filter)

	conn := serveClickFromIP(h, cid, "2001:db8::1")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls)
}

func TestIPv4RotationTable_observe_resetsWindow(t *testing.T) {
	table := NewIPv4RotationTable()
	table.SetMode("live")
	table.SetPolicy(uint64(50*time.Millisecond.Nanoseconds()), 2)

	id := uuidFromBytes(0x03)
	campaignHash := crc32Castagnoli(&id)
	userHash := hashClickUserID("user-a")
	subnet24 := uint32(0xCB007100) // 203.0.113.0
	now := monotonicNano()

	live, shadow := table.observe(campaignHash, userHash, subnet24, 0xCB007101, now)
	require.False(t, live)
	require.False(t, shadow)

	live, shadow = table.observe(campaignHash, userHash, subnet24, 0xCB007102, now)
	require.True(t, live)
	require.False(t, shadow)

	live, shadow = table.observe(campaignHash, userHash, subnet24, 0xCB007103, now+int64(100*time.Millisecond))
	require.False(t, live)
	require.False(t, shadow)
}

func TestIPv4HostAndSubnet24(t *testing.T) {
	host, subnet, ok := ipv4HostAndSubnet24("203.0.113.42")
	require.True(t, ok)
	require.Equal(t, uint32(0xCB00712A), host)
	require.Equal(t, uint32(0xCB007100), subnet)
}

func TestIPv4RotationMetric_registered(t *testing.T) {
	require.NotNil(t, metrics.IPv4RotationMatchTotal)
	require.NotNil(t, metrics.IPv4RotationShadowTotal)
}
