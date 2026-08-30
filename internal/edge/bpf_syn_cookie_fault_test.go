package edge

import (
	"encoding/binary"
	"net"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	xdpDrop = 1
	xdpPass = 2
	xdpTX   = 3
)

func TestFault_XDPSynCookieReturnCode(t *testing.T) {
	objs := loadTestObjects(t)
	if objs.XdpSynCookie == nil {
		t.Skip("xdp_syn_cookie not available in this kernel")
	}

	pkt := buildSYNPacket(t, net.IPv4(192, 0, 2, 1), net.IPv4(10, 0, 0, 1), trackerPort)

	ret, _, err := objs.XdpSynCookie.Test(pkt)
	require.NoError(t, err)

	assert.NotEqual(t, xdpPass, ret, "SYN must not reach kernel stack via XDP_PASS")

	cookies := statCount(t, objs.Stats, StatSynCookie)
	if cookies > 0 {
		assert.Equal(t, xdpTX, ret, "cookie issued must return XDP_TX")
		testutil.LogFaultProof(t, "xdp_syn_cookie_mitigation", map[string]string{
			"status": "pass",
			"return": xdpActionLabel(ret),
		})
		return
	}

	assert.Equal(t, xdpDrop, ret, "helper unavailable must DROP, not PASS")
	testutil.LogFaultProof(t, "xdp_syn_cookie_mitigation", map[string]string{
		"status": "helper_unavailable",
		"return": xdpActionLabel(ret),
	})
}

func TestFault_XDPSynCookieWithTCPOptions(t *testing.T) {
	objs := loadTestObjects(t)
	if objs.XdpSynCookie == nil {
		t.Skip("xdp_syn_cookie not available in this kernel")
	}

	const ethLen = 14
	const ipLen = 20
	const outTCPHLen = 20

	src := net.IPv4(192, 0, 2, 1)
	dst := net.IPv4(10, 0, 0, 1)
	pkt := buildSYNPacketWithMSS(t, src, dst, trackerPort, 65535, 64, 1460)
	require.Len(t, pkt, ethLen+ipLen+24, "SYN with MSS option must be 58-byte frame")

	ret, modifiedPkt, err := objs.XdpSynCookie.Test(pkt)
	require.NoError(t, err)
	if ret != xdpTX {
		assert.Equal(t, xdpDrop, ret, "helper unavailable must DROP")
		t.Skip("syn cookie helper unavailable")
	}

	assert.Len(t, modifiedPkt, ethLen+ipLen+outTCPHLen)
	assert.Equal(t, uint16(ipLen+outTCPHLen), ipv4TotalLen(modifiedPkt))
	assert.Equal(t, byte(5), tcpDataOffset(modifiedPkt))
}

func TestFault_XDPSynCookieWithPayload(t *testing.T) {
	objs := loadTestObjects(t)
	if objs.XdpSynCookie == nil {
		t.Skip("xdp_syn_cookie not available in this kernel")
	}

	const ethLen = 14
	const ipLen = 20
	const outTCPHLen = 20
	const payloadLen = 4

	src := net.IPv4(192, 0, 2, 2)
	dst := net.IPv4(10, 0, 0, 1)
	pkt := buildSYNPacketWithMSSAndPayload(t, src, dst, trackerPort, 65535, 64, 1460, []byte{0xde, 0xad, 0xbe, 0xef})
	require.Len(t, pkt, ethLen+ipLen+24+payloadLen, "SYN with options and payload must be 62-byte frame")

	ret, modifiedPkt, err := objs.XdpSynCookie.Test(pkt)
	require.NoError(t, err)
	if ret != xdpTX {
		assert.Equal(t, xdpDrop, ret, "helper unavailable must DROP")
		t.Skip("syn cookie helper unavailable")
	}

	assert.Len(t, modifiedPkt, ethLen+ipLen+outTCPHLen)
	assert.Equal(t, uint16(ipLen+outTCPHLen), ipv4TotalLen(modifiedPkt))
	assert.Equal(t, byte(5), tcpDataOffset(modifiedPkt))
}

func buildSYNPacketWithMSSAndPayload(t *testing.T, src, dst net.IP, dport, window uint16, ttl byte, mss uint16, payload []byte) []byte {
	t.Helper()
	pkt := buildSYNPacketWithMSS(t, src, dst, dport, window, ttl, mss)
	pkt = append(pkt, payload...)
	ipTotal := uint16(len(pkt) - ethHeaderLen)
	binary.BigEndian.PutUint16(pkt[ethHeaderLen+2:ethHeaderLen+4], ipTotal)
	return pkt
}

const ethHeaderLen = 14

func ipv4TotalLen(pkt []byte) uint16 {
	return binary.BigEndian.Uint16(pkt[ethHeaderLen+2 : ethHeaderLen+4])
}

func tcpDataOffset(pkt []byte) byte {
	const ipLen = 20
	return pkt[ethHeaderLen+ipLen+12] >> 4
}

func xdpActionLabel(act uint32) string {
	switch act {
	case 0:
		return "XDP_ABORTED"
	case xdpDrop:
		return "XDP_DROP"
	case xdpPass:
		return "XDP_PASS"
	case xdpTX:
		return "XDP_TX"
	case 4:
		return "XDP_REDIRECT"
	default:
		return "UNKNOWN"
	}
}
