package edge

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeViolation_ipv4(t *testing.T) {
	raw := make([]byte, ViolationEventWireSize)
	binary.LittleEndian.PutUint64(raw[0:8], 99)
	raw[8] = ViolationAddrFamilyV4
	raw[9] = ViolationSYN
	copy(raw[12:16], net.ParseIP("203.0.113.200").To4())

	evt, ok := decodeViolation(raw)
	require.True(t, ok)
	assert.Equal(t, "203.0.113.200", ViolationHost(evt))
	assert.Equal(t, uint8(ViolationSYN), evt.Reason)
}

func TestDecodeViolation_ipv6(t *testing.T) {
	raw := make([]byte, ViolationEventWireSize)
	binary.LittleEndian.PutUint64(raw[0:8], 42)
	raw[8] = ViolationAddrFamilyV6
	raw[9] = ViolationPPS
	v6 := net.ParseIP("2001:db8:1::42").To16()
	require.NotNil(t, v6)
	copy(raw[12:28], v6)

	evt, ok := decodeViolation(raw)
	require.True(t, ok)
	assert.Equal(t, "2001:db8:1::42", ViolationHost(evt))
	assert.Equal(t, uint8(ViolationPPS), evt.Reason)
}

func TestDecodeViolation_shortSampleRejected(t *testing.T) {
	_, ok := decodeViolation([]byte{1, 2, 3})
	assert.False(t, ok)
}

func TestViolationHandler_dedupesByHost(t *testing.T) {
	v4 := make([]byte, ViolationEventWireSize)
	v4[8] = ViolationAddrFamilyV4
	v4[9] = ViolationSYN
	copy(v4[12:16], net.ParseIP("198.51.100.1").To4())

	v6 := make([]byte, ViolationEventWireSize)
	v6[8] = ViolationAddrFamilyV6
	v6[9] = ViolationSYN
	copy(v6[12:28], net.ParseIP("2001:db8::1").To16())

	seen := make(map[string]struct{})
	var hosts []string
	add := func(raw []byte) {
		evt, ok := decodeViolation(raw)
		require.True(t, ok)
		host := ViolationHost(evt)
		if _, dup := seen[host]; dup {
			return
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}
	add(v4)
	add(v4)
	add(v6)

	assert.Equal(t, []string{"198.51.100.1", "2001:db8::1"}, hosts)
}
