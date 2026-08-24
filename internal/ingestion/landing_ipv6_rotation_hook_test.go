package ingestion

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func ipv6RotationHandler(t *testing.T, cidrBlockEnabled bool, mode string, threshold uint32, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	h, cid := cidrBlockHookHandler(t, cidrBlockEnabled, filter)
	table := NewIPv6RotationTable()
	table.SetMode(mode)
	table.SetPolicy(uint64(time.Minute.Nanoseconds()), threshold)
	h.ConfigureIPv6Rotation(table)
	return h, cid
}

func rotatedIPv6Host(n int) string {
	return fmt.Sprintf("2001:db8:85a3::%x", n)
}

func TestClickRedirect_IPv6Rotation_Live_SafeView(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv6RotationHandler(t, true, "live", 4, filter)

	for i := 1; i <= 3; i++ {
		conn := serveClickFromIP(h, cid, rotatedIPv6Host(i))
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	}
	conn := serveClickFromIP(h, cid, rotatedIPv6Host(4))
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "X-ad-event-processor-Safe-View: l1v6")
	require.Equal(t, 3, filter.calls, "live block must short-circuit before FilterEngine")
}

func TestClickRedirect_IPv6Rotation_Shadow_FallsThrough(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv6RotationHandler(t, true, "shadow", 3, filter)

	for i := 1; i <= 4; i++ {
		conn := serveClickFromIP(h, cid, rotatedIPv6Host(i))
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
		require.Equal(t, i, filter.calls, "shadow must reach FilterEngine on every click")
	}
}

func TestClickRedirect_IPv6Rotation_Shadow_L2Signal(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv6RotationHandler(t, true, "shadow", 2, filter)

	conn := serveClickFromIP(h, cid, rotatedIPv6Host(1))
	require.Equal(t, 1, filter.calls)
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")

	conn = serveClickFromIP(h, cid, rotatedIPv6Host(2))
	require.Equal(t, 2, filter.calls)
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
}

func TestClickRedirect_IPv6Rotation_CampaignDisabled(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv6RotationHandler(t, false, "live", 2, filter)

	for i := 1; i <= 3; i++ {
		conn := serveClickFromIP(h, cid, rotatedIPv6Host(i))
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	}
	require.Equal(t, 3, filter.calls)
}

func TestClickRedirect_IPv6Rotation_IPv4Skipped(t *testing.T) {
	filter := &countingFilter{}
	h, cid := ipv6RotationHandler(t, true, "live", 2, filter)

	conn := serveClickFromIP(h, cid, "8.8.8.8")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls)
}

func TestClickRedirect_IPv6Rotation_TableNil_FailOpen(t *testing.T) {
	filter := &countingFilter{}
	h, cid := cidrBlockHookHandler(t, true, filter)

	for i := 1; i <= 3; i++ {
		conn := serveClickFromIP(h, cid, rotatedIPv6Host(i))
		require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	}
	require.Equal(t, 3, filter.calls)
}

func TestIPv6RotationTable_observe_resetsWindow(t *testing.T) {
	table := NewIPv6RotationTable()
	table.SetMode("live")
	table.SetPolicy(uint64(50*time.Millisecond.Nanoseconds()), 2)

	id := uuidFromBytes(0x01)
	campaignHash := crc32Castagnoli(&id)
	v6Hi := uint64(0x20010db885a30000)
	now := monotonicNano()

	live, shadow := table.observe(campaignHash, v6Hi, 1, now)
	require.False(t, live)
	require.False(t, shadow)

	live, shadow = table.observe(campaignHash, v6Hi, 2, now)
	require.True(t, live)
	require.False(t, shadow)

	live, shadow = table.observe(campaignHash, v6Hi, 3, now+int64(100*time.Millisecond))
	require.False(t, live)
	require.False(t, shadow)
}

func uuidFromBytes(b byte) uuid.UUID {
	var id uuid.UUID
	id[15] = b
	return id
}

func TestIPv6RotationTable_distinctLoOnly(t *testing.T) {
	table := NewIPv6RotationTable()
	table.SetMode("live")
	table.SetPolicy(uint64(time.Minute.Nanoseconds()), 3)

	id := uuidFromBytes(0x02)
	campaignHash := crc32Castagnoli(&id)
	v6Hi := uint64(0x20010db885a30000)
	now := monotonicNano()

	for i := 0; i < 5; i++ {
		live, _ := table.observe(campaignHash, v6Hi, 0xabc, now)
		require.False(t, live)
	}
	live, _ := table.observe(campaignHash, v6Hi, 0xdef, now)
	require.False(t, live)
	live, _ = table.observe(campaignHash, v6Hi, 0x123, now)
	require.True(t, live)
}
