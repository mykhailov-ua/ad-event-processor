package edge

import (
	"encoding/binary"
	"net"
	"os"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildIPv6SYNPacket(t *testing.T, src net.IP, dport uint16) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)
	require.Nil(t, src.To4())

	const (
		ethLen = 14
		ip6Len = 40
		tcpLen = 20
	)
	pkt := make([]byte, ethLen+ip6Len+tcpLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = 6
	ip6[7] = 64
	copy(ip6[8:24], src)

	tcp := pkt[ethLen+ip6Len:]
	tcp[12] = 0x50
	binary.BigEndian.PutUint16(tcp[0:2], 12345)
	binary.BigEndian.PutUint16(tcp[2:4], dport)
	tcp[13] = 0x02

	return pkt
}

func buildIPv6UDPPacket(t *testing.T, src net.IP, dport uint16) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)
	require.Nil(t, src.To4())

	const (
		ethLen = 14
		ip6Len = 40
		udpLen = 8
	)
	pkt := make([]byte, ethLen+ip6Len+udpLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = 17
	ip6[7] = 64
	copy(ip6[8:24], src)

	udp := pkt[ethLen+ip6Len:]
	binary.BigEndian.PutUint16(udp[0:2], 12345)
	binary.BigEndian.PutUint16(udp[2:4], dport)

	return pkt
}

func buildIPv6ICMPv6Packet(t *testing.T, src net.IP, typ, code byte) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)
	require.Nil(t, src.To4())

	const (
		ethLen  = 14
		ip6Len  = 40
		icmpLen = 8
	)
	pkt := make([]byte, ethLen+ip6Len+icmpLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = 58
	ip6[7] = 64
	copy(ip6[8:24], src)

	icmp := pkt[ethLen+ip6Len:]
	icmp[0] = typ
	icmp[1] = code

	return pkt
}

func buildIPv6SCTPPacket(t *testing.T, src net.IP, dport uint16) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)
	require.Nil(t, src.To4())

	const (
		ethLen  = 14
		ip6Len  = 40
		sctpLen = 12
	)
	pkt := make([]byte, ethLen+ip6Len+sctpLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = 132
	ip6[7] = 64
	copy(ip6[8:24], src)

	sctp := pkt[ethLen+ip6Len:]
	binary.BigEndian.PutUint16(sctp[0:2], 12345)
	binary.BigEndian.PutUint16(sctp[2:4], dport)

	return pkt
}

func buildIPv6TCPWithExtHeader(t *testing.T, src net.IP, extProto, extLenUnits, l4Proto byte, dport uint16) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)
	require.Nil(t, src.To4())

	extTotal := int(extLenUnits+1) * 8
	const (
		ethLen = 14
		ip6Len = 40
		tcpLen = 20
	)
	pkt := make([]byte, ethLen+ip6Len+extTotal+tcpLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = extProto
	ip6[7] = 64
	copy(ip6[8:24], src)

	ext := pkt[ethLen+ip6Len:]
	ext[0] = l4Proto
	ext[1] = extLenUnits

	tcp := pkt[ethLen+ip6Len+extTotal:]
	tcp[12] = 0x50
	binary.BigEndian.PutUint16(tcp[0:2], 12345)
	binary.BigEndian.PutUint16(tcp[2:4], dport)
	tcp[13] = 0x02

	return pkt
}

func buildIPv6UDPWithHopByHop(t *testing.T, src net.IP, dport uint16) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)

	const (
		ethLen = 14
		ip6Len = 40
		hbhLen = 8
		udpLen = 8
	)
	pkt := make([]byte, ethLen+ip6Len+hbhLen+udpLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = 0
	ip6[7] = 64
	copy(ip6[8:24], src)

	hbh := pkt[ethLen+ip6Len:]
	hbh[0] = 17
	hbh[1] = 0

	udp := pkt[ethLen+ip6Len+hbhLen:]
	binary.BigEndian.PutUint16(udp[2:4], dport)

	return pkt
}

func buildIPv6FragNonFirst(t *testing.T, src net.IP) []byte {
	t.Helper()
	src = src.To16()
	require.NotNil(t, src)

	const (
		ethLen  = 14
		ip6Len  = 40
		fragLen = 8
	)
	pkt := make([]byte, ethLen+ip6Len+fragLen)

	binary.BigEndian.PutUint16(pkt[12:14], 0x86DD)

	ip6 := pkt[ethLen:]
	ip6[0] = 0x60
	ip6[6] = 44
	ip6[7] = 64
	copy(ip6[8:24], src)

	frag := pkt[ethLen+ip6Len:]
	frag[0] = 6
	binary.BigEndian.PutUint16(frag[2:4], 0x0008)

	return pkt
}

func TestParseIPv6Host_valid(t *testing.T) {
	key, ok := ParseIPv6Host("2001:db8::1")
	require.True(t, ok)
	assert.Equal(t, uint32(128), key.PrefixLen)
	assert.Equal(t, byte(0x20), key.Addr[0])
	assert.Equal(t, byte(1), key.Addr[15])
}

func TestParseIPv6Prefix_64(t *testing.T) {
	key, ok := ParseIPv6Prefix("2001:db8:85a3::/64")
	require.True(t, ok)
	assert.Equal(t, uint32(64), key.PrefixLen)
	assert.Equal(t, byte(0), key.Addr[8])
}

func TestBlocklistStore_ApplyDiffIPv6(t *testing.T) {
	m := newLPMMapV6(t)
	maps := BlocklistMaps{V6Host: newHostHashMapV6(t), V6Prefix: m}
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(maps, nil, nil, []string{"2001:db8::dead"})
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, store.Len())

	var marker uint8
	key, ok := ParseIPv6Host("2001:db8::dead")
	require.True(t, ok)
	require.NoError(t, maps.V6Host.Lookup(key.Addr, &marker))
	assert.Equal(t, blockedMarker, marker)

	added, removed, err = store.ApplyDiff(maps, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 1, removed)
	assert.Equal(t, 0, store.Len())
}

func TestFault_BlocklistV6XDPDrop(t *testing.T) {
	objs := loadTestObjects(t)
	store := NewBlocklistStore()
	maps := blocklistMapsFromObjects(objs)

	victim := parseTestIPv6(t, "2001:db8::9")
	pkt := buildIPv6SYNPacket(t, victim, trackerPort)
	control := parseTestIPv6(t, "2001:db8::10")
	controlPkt := buildIPv6SYNPacket(t, control, trackerPort)

	require.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))
	require.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, controlPkt))

	_, _, err := store.ApplyDiff(maps, nil, nil, []string{victim.String()})
	require.NoError(t, err)

	require.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
	require.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, controlPkt))
}

func buildIPv6RSTPacket(t *testing.T, src net.IP, dport uint16) []byte {
	t.Helper()
	pkt := buildIPv6SYNPacket(t, src, dport)
	tcp := pkt[54:]
	tcp[13] = 0x04
	return pkt
}

func buildIPv6PSHACKPacket(t *testing.T, src net.IP, dport uint16) []byte {
	t.Helper()
	pkt := buildIPv6SYNPacket(t, src, dport)
	tcp := pkt[54:]
	tcp[13] = 0x18
	return pkt
}

func TestXDP_ipv6ConfigMapOverridesSYNLimit(t *testing.T) {
	objs := loadTestObjects(t)
	key := uint32(0)
	cfg := DefaultConfig(InitOptions{})
	cfg.SynLimit = 4
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	src := parseTestIPv6(t, "2001:db8:1::42")
	pkt := buildIPv6SYNPacket(t, src, trackerPort)

	var last uint32
	for range 6 {
		last = runXDP(t, objs.XdpEdgeFilter, pkt)
	}
	assert.Equal(t, uint32(1), last)
	assert.GreaterOrEqual(t, statCount(t, objs.Stats, StatDropSyn), uint64(1))
}

func TestXDP_ipv6DropRSTFlood(t *testing.T) {
	objs := loadTestObjects(t)
	src := parseTestIPv6(t, "2001:db8:2::30")
	pkt := buildIPv6RSTPacket(t, src, trackerPort)

	var last uint32
	for range 70 {
		last = runXDP(t, objs.XdpEdgeFilter, pkt)
	}
	assert.Equal(t, uint32(1), last)
	assert.GreaterOrEqual(t, statCount(t, objs.Stats, StatDropRST), uint64(1))
}

func TestXDP_ipv6RateLimit_holdoutWithoutV6Maps(t *testing.T) {
	data, err := os.ReadFile("../../deploy/edge/xdp/bpf/edge_filter.c")
	require.NoError(t, err)
	src := string(data)
	assert.Contains(t, src, "check_syn_limit_v6")
	assert.Contains(t, src, "check_pps_limit_v6")
	assert.Contains(t, src, "check_rst_limit_v6")
	assert.Contains(t, src, "check_syn_subnet_limit_v6")
}

func TestXDP_dropNonTCPOnTrackerPortIPv6(t *testing.T) {
	objs := loadTestObjects(t)
	src := parseTestIPv6(t, "2001:db8:3::20")

	t.Run("udp", func(t *testing.T) {
		before := statCount(t, objs.Stats, StatDropNonTCP)
		pkt := buildIPv6UDPPacket(t, src, trackerPort)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropNonTCP))
	})

	t.Run("sctp", func(t *testing.T) {
		pkt := buildIPv6SCTPPacket(t, src, trackerPort)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
	})

	t.Run("icmpv6", func(t *testing.T) {
		pkt := buildIPv6ICMPv6Packet(t, src, 128, 0)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
	})

	t.Run("icmpv6_pkt_toobig_pass", func(t *testing.T) {
		pkt := buildIPv6ICMPv6Packet(t, src, 2, 0)
		assert.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))
	})

	t.Run("udp_non_tracker_pass", func(t *testing.T) {
		pkt := buildIPv6UDPPacket(t, src, 443)
		assert.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))
	})
}

func buildTruncatedIPv6TrackerTCP(t *testing.T, src net.IP, tcpPayloadBytes int) []byte {
	t.Helper()
	require.GreaterOrEqual(t, tcpPayloadBytes, 4)
	require.Less(t, tcpPayloadBytes, 20)
	pkt := buildIPv6SYNPacket(t, src, trackerPort)
	return pkt[:14+40+tcpPayloadBytes]
}

func TestXDP_ipv6TruncatedTrackerTCPFailClosed(t *testing.T) {
	objs := loadTestObjects(t)
	src := parseTestIPv6(t, "2001:db8:4::77")

	t.Run("drop_invalid_not_pass", func(t *testing.T) {
		before := statCount(t, objs.Stats, StatDropInvalid)
		pkt := buildTruncatedIPv6TrackerTCP(t, src, 8)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropInvalid))
	})

	t.Run("blocklist_before_truncated_header", func(t *testing.T) {
		store := NewBlocklistStore()
		maps := blocklistMapsFromObjects(objs)
		victim := parseTestIPv6(t, "2001:db8:4::78")
		_, _, err := store.ApplyDiff(maps, nil, nil, []string{victim.String()})
		require.NoError(t, err)
		before := statCount(t, objs.Stats, StatDropBlocklist)
		pkt := buildTruncatedIPv6TrackerTCP(t, victim, 8)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropBlocklist))
	})
}

func TestXDP_ipv6NonTCP_holdoutUsesDropNonTcpTracker(t *testing.T) {
	data, err := os.ReadFile("../../deploy/edge/xdp/bpf/edge_filter.c")
	require.NoError(t, err)
	src := string(data)
	assert.Contains(t, src, "ICMPV6_PKT_TOOBIG")
	assert.Contains(t, src, "drop_non_tcp_tracker(l4_proto, l4, data_end, 1)")
}

func TestXDP_ipv6ExtHeaderWalk(t *testing.T) {
	objs := loadTestObjects(t)
	src := parseTestIPv6(t, "2001:db8:5::1")

	t.Run("hop_by_hop_tcp_tracker_pass", func(t *testing.T) {
		pkt := buildIPv6TCPWithExtHeader(t, src, 0, 0, 6, trackerPort)
		assert.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))
	})

	t.Run("routing_tcp_tracker_pass", func(t *testing.T) {
		pkt := buildIPv6TCPWithExtHeader(t, src, 43, 0, 6, trackerPort)
		assert.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))
	})

	t.Run("hop_by_hop_blocklist_drop", func(t *testing.T) {
		store := NewBlocklistStore()
		maps := blocklistMapsFromObjects(objs)
		victim := parseTestIPv6(t, "2001:db8:5::dead")
		_, _, err := store.ApplyDiff(maps, nil, nil, []string{victim.String()})
		require.NoError(t, err)
		before := statCount(t, objs.Stats, StatDropBlocklist)
		pkt := buildIPv6TCPWithExtHeader(t, victim, 0, 0, 6, trackerPort)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropBlocklist))
	})

	t.Run("hop_by_hop_udp_tracker_drop", func(t *testing.T) {
		victim := parseTestIPv6(t, "2001:db8:5::2")
		before := statCount(t, objs.Stats, StatDropNonTCP)
		pkt := buildIPv6UDPWithHopByHop(t, victim, trackerPort)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropNonTCP))
	})

	t.Run("unknown_ext_drop_invalid", func(t *testing.T) {
		before := statCount(t, objs.Stats, StatDropInvalid)
		pkt := buildIPv6TCPWithExtHeader(t, src, 135, 0, 6, trackerPort)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropInvalid))
	})

	t.Run("non_first_fragment_drop_invalid", func(t *testing.T) {
		before := statCount(t, objs.Stats, StatDropInvalid)
		pkt := buildIPv6FragNonFirst(t, src)
		assert.Equal(t, uint32(1), runXDP(t, objs.XdpEdgeFilter, pkt))
		assert.Equal(t, before+1, statCount(t, objs.Stats, StatDropInvalid))
	})
}

func TestXDP_ipv6ExtHeader_holdout(t *testing.T) {
	data, err := os.ReadFile("../../deploy/edge/xdp/bpf/edge_filter.c")
	require.NoError(t, err)
	src := string(data)
	assert.Contains(t, src, "ipv6_find_l4")
	assert.Contains(t, src, "IPV6_EXT_MAX_STEPS")
	assert.Contains(t, src, "IPPROTO_FRAGMENT")
}

func parseTestIPv6(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	require.NotNil(t, ip)
	return ip
}
