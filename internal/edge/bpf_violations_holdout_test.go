package edge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViolationEmit_noRandomSampling_holdout(t *testing.T) {
	data, err := os.ReadFile("../../deploy/edge/xdp/bpf/edge_filter.c")
	require.NoError(t, err)
	src := string(data)
	assert.NotContains(t, src, "RINGBUF_VIOLATION_PPS_SAMPLE_PCT")
	assert.NotContains(t, src, "RINGBUF_VIOLATION_OTHER_SAMPLE_PCT")
	assert.NotContains(t, src, "(src_ip ^ reason) & 0x7")
	assert.NotContains(t, src, "(src_ip ^ reason) & 0x3")
	assert.Contains(t, src, "RINGBUF_VIOLATION_PPS_STOP_PCT")
}

func TestViolationEmit_ipv6AutobanWire_holdout(t *testing.T) {
	data, err := os.ReadFile("../../deploy/edge/xdp/bpf/edge_filter.c")
	require.NoError(t, err)
	src := string(data)
	assert.Contains(t, src, "emit_violation_v6")
	assert.Contains(t, src, "addr_family")
	assert.Contains(t, src, "__u8 addr[16]")
	assert.Contains(t, src, "emit_violation_v6(&ip6->saddr, VIOLATION_PPS)")
}
