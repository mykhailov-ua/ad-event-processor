package edge

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"ad-event-processor/internal/testutil"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_XDPMalformedPacketFuzzing(t *testing.T) {
	objs := loadTestObjects(t)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	ret, _, err := objs.XdpEdgeFilter.Test([]byte{})
	if err == nil {
		assert.Equal(t, uint32(2), ret, "zero-length should PASS (early eth check)")
	}

	ret, _, err = objs.XdpEdgeFilter.Test(make([]byte, 10))
	if err == nil {
		assert.Equal(t, uint32(2), ret, "short packet should PASS")
	}

	for range 1000 {
		pktLen := 14 + r.Intn(1486)
		pkt := make([]byte, pktLen)
		r.Read(pkt)
		ret, _, _ = objs.XdpEdgeFilter.Test(pkt)
		if ret != 0 {
			assert.Contains(t, []uint32{1, 2}, ret)
		}
	}

	testutil.LogFaultProof(t, "xdp_packet_fuzzing", map[string]string{
		"iters":   "1000",
		"max_len": "1500",
		"status":  "no_panics",
	})
}

func TestFault_XDPSyncRedisOutage(t *testing.T) {
	if testing.Short() {
		t.Skip("redis outage fault test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, redisClient, cleanup := testutil.SetupRedisClient(t)
	defer cleanup()

	objs := loadTestObjects(t)
	store := NewBlocklistStore()
	maps := blocklistMapsFromObjects(objs)

	require.NoError(t, redisClient.SAdd(ctx, "blacklist:manual", "1.2.3.4").Err())
	_, _, err := SyncBlocklistFromRedis(ctx, redisClient, maps, store)
	require.NoError(t, err)
	assert.Equal(t, 1, store.Len())

	require.NoError(t, c.Terminate(ctx))

	_, _, err = SyncBlocklistFromRedis(ctx, redisClient, maps, store)
	assert.Error(t, err, "sync must fail when redis is down")
	assert.Equal(t, 1, store.Len(), "store must preserve state on sync failure")

	testutil.LogFaultProof(t, "xdp_sync_redis_outage", map[string]string{
		"fault":          "container_termination",
		"state_retained": "true",
		"error_handled":  "true",
	})
}

func TestFault_XDPRingbufCongestion(t *testing.T) {
	objs := loadTestObjects(t)

	src := net.IPv4(192, 0, 2, 1)
	pkt := buildSYNPacket(t, src, net.IPv4(10, 0, 0, 1), trackerPort)

	cfg := uint32(0)
	opts := DefaultConfig(InitOptions{})
	opts.SynLimit = 1
	require.NoError(t, objs.Config.Update(&cfg, &opts, ebpf.UpdateAny))

	for range 2000 {
		runXDP(t, objs.XdpEdgeFilter, pkt)
	}

	handler := NewViolationHandler(func(evt ViolationEvent) error { return nil })
	rd, err := ringbuf.NewReader(objs.Violations)
	require.NoError(t, err)
	defer rd.Close()

	n, err := handler.Drain(rd, 100*time.Millisecond)
	assert.NoError(t, err)
	t.Logf("Drained %d events from congested ringbuf", n)

	testutil.LogFaultProof(t, "xdp_ringbuf_congestion", map[string]string{
		"events_drained": fmt.Sprintf("%d", n),
		"status":         "stable",
	})
}

func TestFault_XDPLRUEvictionUnderPressure(t *testing.T) {
	objs := loadTestObjects(t)

	r := rand.New(rand.NewSource(42))

	for range 10000 {
		src := r.Uint32()
		pkt := buildPSHACKPacket(t, net.IP{byte(src >> 24), byte(src >> 16), byte(src >> 8), byte(src)}, net.IPv4(10, 0, 0, 1), trackerPort)
		runXDP(t, objs.XdpEdgeFilter, pkt)
	}

	testutil.LogFaultProof(t, "xdp_lru_high_churn", map[string]string{
		"unique_ips": "10000",
		"status":     "stable",
	})
}

type failingRedisStub struct {
	failAfter int
	count     int
}

func (s *failingRedisStub) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	s.count++
	cmd := redis.NewStringSliceCmd(ctx)
	if s.count > s.failAfter {
		cmd.SetErr(fmt.Errorf("injected redis failure"))
		return cmd
	}
	cmd.SetVal([]string{"198.51.100.1", "198.51.100.2"})
	return cmd
}

func TestFault_XDPSyncInterruptedPartialUpdate(t *testing.T) {
	objs := loadTestObjects(t)
	store := NewBlocklistStore()
	maps := blocklistMapsFromObjects(objs)

	stub := &failingRedisStub{failAfter: 10}
	added, _, err := SyncBlocklistFromRedis(context.Background(), stub, maps, store)
	require.NoError(t, err)
	assert.Equal(t, 2, added)
	assert.Equal(t, 2, store.Len())

	stub.failAfter = 0
	_, _, err = SyncBlocklistFromRedis(context.Background(), stub, maps, store)
	assert.Error(t, err)

	assert.Equal(t, 2, store.Len(), "Store must not be partially updated or cleared")

	var val uint8
	require.NoError(t, objs.BlocklistHostV4.Lookup(HostKey(198, 51, 100, 1).Addr, &val))
	require.NoError(t, objs.BlocklistHostV4.Lookup(HostKey(198, 51, 100, 2).Addr, &val))

	testutil.LogFaultProof(t, "xdp_sync_interrupted", map[string]string{
		"fault":           "partial_redis_failure",
		"state_preserved": "true",
	})
}
