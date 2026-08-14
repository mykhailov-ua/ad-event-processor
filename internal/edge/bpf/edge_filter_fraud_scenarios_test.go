package bpf

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/testutil"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraudScenarios_X06_HighVolumeSubnetBurstDrops(t *testing.T) {
	if testing.Short() {
		t.Skip("requires CAP_BPF")
	}
	objs := loadTestObjects(t)

	key := uint32(0)
	cfg := DefaultConfig(InitOptions{})
	cfg.SynSubnetLimit = 4
	cfg.SynLimit = 1000
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	var drops uint64
	for h := range 8 {
		src := net.IPv4(203, 0, 113, byte(h+1))
		pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
		for range 3 {
			if runXDP(t, objs.XdpEdgeFilter, pkt) == 1 {
				drops++
			}
		}
	}
	require.Greater(t, drops, uint64(0), "tuned subnet cap must drop high-volume /24 burst")
	assert.GreaterOrEqual(t, statCount(t, objs.Stats, StatDropSynSubnet), uint64(1))

	testutil.LogFaultProof(t, "xdp_fraud_x06_high_volume_subnet_drop", map[string]string{
		"syn_subnet_limit": "4",
		"drops":            fmt.Sprintf("%d", drops),
		"status":           "enforced",
	})
}

func TestFraudScenarios_X06_LowVolumeRotationAcceptedGap(t *testing.T) {
	if testing.Short() {
		t.Skip("requires CAP_BPF")
	}
	objs := loadTestObjects(t)

	const hosts = 32
	var drops uint64
	for h := range hosts {
		src := net.IPv4(203, 0, 113, byte(h+1))
		pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
		if runXDP(t, objs.XdpEdgeFilter, pkt) == 1 {
			drops++
		}
	}
	// Product decision: accept low-volume /24 rotation under default syn_subnet_limit (256).
	// Backstop: Nginx Lua rate limit, tracker filters, IVT + fraud-scorer (see EDGE_XDP.md).
	require.Equal(t, uint64(0), drops, "one SYN per host must stay under default subnet cap")

	testutil.LogFaultProof(t, "xdp_fraud_x06_low_volume_rotation", map[string]string{
		"hosts":    fmt.Sprintf("%d", hosts),
		"drops":    "0",
		"decision": "accepted",
		"backstop": "lua_ivt_fraud_scorer",
	})
}

func TestFraudScenarios_X07_FingerprintEnabledEmits(t *testing.T) {
	if testing.Short() {
		t.Skip("requires CAP_BPF")
	}
	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	key := uint32(0)
	cfg := DefaultConfig(InitOptions{})
	cfg.FingerprintEnabled = 1
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	rd, err := ringbuf.NewReader(objs.Fingerprints)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Close() })

	handler := NewFingerprintHandler(func(evt FingerprintEvent) error { return nil })

	src := net.IPv4(198, 51, 100, 42)
	pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
	require.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))

	n, err := handler.Drain(rd, 500*time.Millisecond)
	require.NoError(t, err)
	require.Greater(t, n, 0, "fingerprint ringbuf must emit when XDP_FINGERPRINT enabled")

	testutil.LogFaultProof(t, "xdp_fraud_x07_fingerprint_enabled", map[string]string{
		"events": fmt.Sprintf("%d", n),
		"status": "emitted",
	})
}

func TestFraudScenarios_X07_FingerprintDisabledAcceptedGap(t *testing.T) {
	if testing.Short() {
		t.Skip("requires CAP_BPF")
	}
	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	key := uint32(0)
	cfg := DefaultConfig(InitOptions{DisableFingerprint: true})
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	rd, err := ringbuf.NewReader(objs.Fingerprints)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Close() })

	handler := NewFingerprintHandler(func(evt FingerprintEvent) error { return nil })

	src := net.IPv4(198, 51, 100, 43)
	pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
	require.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))

	n, err := handler.Drain(rd, 200*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, 0, n, "fingerprint disabled must not emit")

	testutil.LogFaultProof(t, "xdp_fraud_x07_fingerprint_disabled", map[string]string{
		"events":   "0",
		"decision": "accepted",
		"backstop": "ivt_lua_tracker",
	})
}

func TestFraudScenarios_X04_SpoofedSYNStillHandled(t *testing.T) {
	if testing.Short() {
		t.Skip("requires CAP_BPF")
	}
	objs := loadTestObjects(t)
	spoofed := net.IPv4(1, 2, 3, 4)
	pkt := buildSYNPacket(t, spoofed, net.IPv4(10, 0, 0, 1), trackerPort)
	ret := runXDP(t, objs.XdpEdgeFilter, pkt)
	assert.Contains(t, []uint32{1, 2}, ret)
	t.Logf("X-04: spoofed SYN action=%d (RPF not modeled in userspace test)", ret)
}
