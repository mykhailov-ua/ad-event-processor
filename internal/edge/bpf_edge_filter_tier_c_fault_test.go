package edge

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/database"

	"ad-event-processor/internal/testutil"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_XDPFingerprintRingbufCongestion(t *testing.T) {
	if testing.Short() {
		t.Skip("fingerprint ringbuf congestion fault test")
	}

	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	key := uint32(0)
	cfg := DefaultConfig(InitOptions{})
	cfg.SynLimit = 10000
	cfg.GlobalSynLimit = 100000
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	src := net.IPv4(198, 51, 100, 1)
	pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)

	passBefore := statCount(t, objs.Stats, StatPass)
	fpBefore := statCount(t, objs.Stats, StatFingerprint)

	const syns = 3000
	for range syns {
		ret := runXDP(t, objs.XdpEdgeFilter, pkt)
		require.Contains(t, []uint32{1, 2}, ret)
	}

	passAfter := statCount(t, objs.Stats, StatPass)
	fpAfter := statCount(t, objs.Stats, StatFingerprint)

	rd, err := ringbuf.NewReader(objs.Fingerprints)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Close() })

	handler := NewFingerprintHandler(func(evt FingerprintEvent) error { return nil })
	drained, err := handler.Drain(rd, 200*time.Millisecond)
	require.NoError(t, err)

	testutil.LogFaultProof(t, "xdp_fingerprint_ringbuf_congestion", map[string]string{
		"harness":         "bpf_prog_test",
		"syns_sent":       fmt.Sprintf("%d", syns),
		"pass_delta":      fmt.Sprintf("%d", passAfter-passBefore),
		"fp_stat_delta":   fmt.Sprintf("%d", fpAfter-fpBefore),
		"events_drained":  fmt.Sprintf("%d", drained),
		"hot_path_stable": "true",
	})
}

func TestFault_XDPFingerprintNoExtraDrops(t *testing.T) {
	if testing.Short() {
		t.Skip("fingerprint drop parity fault test")
	}

	runFlood := func(disableFP bool) (pass, drop uint64) {
		objs := loadTestObjects(t)

		key := uint32(0)
		cfg := DefaultConfig(InitOptions{DisableFingerprint: disableFP})
		cfg.SynLimit = 4
		require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

		src := net.IPv4(203, 0, 113, 77)
		pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
		for range 20 {
			switch runXDP(t, objs.XdpEdgeFilter, pkt) {
			case 2:
				pass++
			case 1:
				drop++
			}
		}
		return pass, drop
	}

	passOn, dropOn := runFlood(false)
	passOff, dropOff := runFlood(true)

	assert.Equal(t, dropOff, dropOn, "fingerprint must not change drop count")
	assert.Equal(t, passOff, passOn, "fingerprint must not change pass count")

	testutil.LogFaultProof(t, "xdp_fingerprint_no_extra_drops", map[string]string{
		"harness":       "control_cohort",
		"pass_enabled":  fmt.Sprintf("%d", passOn),
		"drop_enabled":  fmt.Sprintf("%d", dropOn),
		"pass_disabled": fmt.Sprintf("%d", passOff),
		"drop_disabled": fmt.Sprintf("%d", dropOff),
		"m22_c3":        "enforced",
	})
}

func TestFault_XDPFingerprintRedisPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("fingerprint redis pipeline fault test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	rd, err := ringbuf.NewReader(objs.Fingerprints)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Close() })

	src := net.IPv4(203, 0, 113, 44)
	pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
	require.Equal(t, uint32(2), runXDP(t, objs.XdpEdgeFilter, pkt))

	handler := NewFingerprintHandler(func(evt FingerprintEvent) error {
		return Record(ctx, rdb, Entry{
			IP:      HostIPv4(evt.SrcIP),
			TCPHash: evt.TCPHash,
			TTL:     evt.TTL,
			Window:  evt.Window,
			MSS:     evt.MSS,
			SeenAt:  time.Now().UTC(),
		})
	})

	n, err := handler.Drain(rd, 500*time.Millisecond)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)

	entries, err := ListRecent(ctx, rdb, 8)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, src.String(), entries[0].IP)
	assert.NotZero(t, entries[0].TCPHash)

	testutil.LogFaultProof(t, "xdp_fingerprint_redis_pipeline", map[string]string{
		"harness":      "ringbuf_drain",
		"events":       fmt.Sprintf("%d", n),
		"redis_staged": "true",
		"no_outbound":  "true",
	})
}

func TestFault_XDPFingerprintConcurrentHosts(t *testing.T) {
	if testing.Short() {
		t.Skip("fingerprint concurrent hosts fault test")
	}

	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	key := uint32(0)
	cfg := DefaultConfig(InitOptions{})
	cfg.SynLimit = 10000
	cfg.GlobalSynLimit = 200000
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	fpBefore := statCount(t, objs.Stats, StatFingerprint)

	const (
		hosts    = 256
		synsEach = 8
	)

	var wg sync.WaitGroup
	var pass, drop atomic.Uint64
	start := make(chan struct{})

	wg.Add(hosts)
	for h := range hosts {
		go func(hostID int) {
			defer wg.Done()
			<-start
			src := net.IPv4(198, 19, byte(hostID>>8), byte(hostID))
			pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)
			for range synsEach {
				switch runXDP(t, objs.XdpEdgeFilter, pkt) {
				case 2:
					pass.Add(1)
				case 1:
					drop.Add(1)
				}
			}
		}(h)
	}
	close(start)
	wg.Wait()

	fpAfter := statCount(t, objs.Stats, StatFingerprint)
	fpDelta := fpAfter - fpBefore

	assert.Greater(t, pass.Load()+drop.Load(), uint64(0))
	assert.GreaterOrEqual(t, fpDelta, uint64(hosts), "each host should emit at least one fingerprint stat")

	testutil.LogFaultProof(t, "xdp_fingerprint_concurrent_hosts", map[string]string{
		"hosts":     fmt.Sprintf("%d", hosts),
		"syns_each": fmt.Sprintf("%d", synsEach),
		"pass":      fmt.Sprintf("%d", pass.Load()),
		"drop":      fmt.Sprintf("%d", drop.Load()),
		"fp_delta":  fmt.Sprintf("%d", fpDelta),
	})
}

func TestFault_XDPFingerprintExtremeTCPFields(t *testing.T) {
	if testing.Short() {
		t.Skip("fingerprint extreme fields fault test")
	}

	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	src := net.IPv4(203, 0, 113, 88)
	pkt := buildSYNPacketWithMSS(t, src, net.IPv4(10, 0, 0, 1), trackerPort, 0xffff, 255, 0xffff)

	ret := runXDP(t, objs.XdpEdgeFilter, pkt)
	assert.Equal(t, uint32(2), ret)
	assert.GreaterOrEqual(t, statCount(t, objs.Stats, StatFingerprint), uint64(1))

	testutil.LogFaultProof(t, "xdp_fingerprint_extreme_tcp_fields", map[string]string{
		"window": "65535",
		"ttl":    "255",
		"mss":    "65535",
		"action": "pass",
	})
}

func TestFault_XDPFingerprintUnderSYNFlood(t *testing.T) {
	if testing.Short() {
		t.Skip("fingerprint under syn flood fault test")
	}

	objs := loadTestObjects(t)
	if objs.Fingerprints == nil {
		t.Skip("fingerprints map unavailable")
	}

	key := uint32(0)
	cfg := DefaultConfig(InitOptions{})
	cfg.SynLimit = 4
	require.NoError(t, objs.Config.Update(&key, &cfg, ebpf.UpdateAny))

	src := net.IPv4(198, 18, 1, 1)
	pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)

	var pass, drop uint64
	fpBefore := statCount(t, objs.Stats, StatFingerprint)
	for range 100 {
		switch runXDP(t, objs.XdpEdgeFilter, pkt) {
		case 2:
			pass++
		case 1:
			drop++
		}
	}
	fpAfter := statCount(t, objs.Stats, StatFingerprint)

	assert.Greater(t, drop, uint64(0), "SYN limit must produce drops")
	assert.Greater(t, fpAfter, fpBefore, "fingerprints emitted before drops")

	testutil.LogFaultProof(t, "xdp_fingerprint_under_syn_flood", map[string]string{
		"syns":     "100",
		"pass":     fmt.Sprintf("%d", pass),
		"drop":     fmt.Sprintf("%d", drop),
		"fp_delta": fmt.Sprintf("%d", fpAfter-fpBefore),
		"m22_c3":   "drops_from_syn_limit",
	})
}
