package edge

import (
	"encoding/binary"
	"net"
	"testing"

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

func parseTestIPv6(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	require.NotNil(t, ip)
	return ip
}
