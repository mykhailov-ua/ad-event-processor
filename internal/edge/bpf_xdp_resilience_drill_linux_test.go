//go:build linux

package edge_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/cilium/ebpf/link"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestResilienceDrill_LoopbackBlocklistDrop(t *testing.T) {
	if os.Getenv("XDP_RESILIENCE_DRILL") != "1" {
		t.Skip("set XDP_RESILIENCE_DRILL=1 to run enterprise XDP drill")
	}
	if os.Geteuid() != 0 {
		t.Skip("XDP resilience drill requires root for lo attach")
	}
	if _, err := os.Stat("/sys/kernel/btf/vmlinux"); err != nil {
		t.Skip("BTF vmlinux required for XDP resilience drill")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	redisAddr := strings.TrimSpace(os.Getenv("REDIS_ADDRS"))
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	redisAddr = strings.Split(redisAddr, ",")[0]

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASS"),
	})
	defer func() { _ = rdb.Close() }()
	require.NoError(t, rdb.Ping(ctx).Err())

	var objs edge.EdgeObjects
	require.NoError(t, edge.LoadEdgeObjectsLenient(&objs, nil))
	defer objs.Close()
	require.NoError(t, edge.InitConfigWith(objs.Config, edge.InitOptions{}))

	iface, err := net.InterfaceByName("lo")
	require.NoError(t, err)

	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.XdpEdgeFilter,
		Interface: iface.Index,
		Flags:     link.XDPGenericMode,
	})
	require.NoError(t, err)
	defer func() { _ = xdpLink.Close() }()

	victim := net.IPv4(203, 0, 113, 150)
	require.NoError(t, rdb.Del(ctx, "blacklist:manual").Err())
	require.NoError(t, rdb.SAdd(ctx, "blacklist:manual", victim.String()).Err())

	store := edge.NewBlocklistStore()
	_, _, err = edge.SyncBlocklistFromRedis(ctx, rdb, objs.BlocklistV4, nil, store)
	require.NoError(t, err)
	require.GreaterOrEqual(t, store.Len(), 1)

	statsBefore, err := edge.AggregateStats(objs.Stats)
	require.NoError(t, err)
	before := statsBefore[edge.StatDropBlocklist]

	pkt := edge.BuildSYNPacket(victim, net.IPv4(10, 0, 0, 1), edge.TrackerPort)
	for range 50 {
		ret, _, err := objs.XdpEdgeFilter.Test(pkt)
		require.NoError(t, err)
		require.Equal(t, uint32(1), ret, "blocklisted source must XDP_DROP")
	}

	statsAfter, err := edge.AggregateStats(objs.Stats)
	require.NoError(t, err)
	after := statsAfter[edge.StatDropBlocklist]
	require.Greater(t, after, before, "blocklist drop counter must increase after prog.Test")

	delta := after - before
	faultproof.Log(t, "xdp_resilience_drill", map[string]string{
		"iface":           "lo",
		"attach":          "generic",
		"drop_assertion":  "prog_test_same_maps",
		"blocklist_drops": fmt.Sprintf("%d", delta),
		"sync_entries":    fmt.Sprintf("%d", store.Len()),
		"status":          "passed",
	})
}
