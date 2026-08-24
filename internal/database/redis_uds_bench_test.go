//go:build !short

package database

import (
	"context"
	"net"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
	_ "unsafe"

	"ad-event-processor/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

//go:linkname monotonicNano runtime.nanotime
func monotonicNano() int64

const (
	udsDialSamples       = 10_000
	udsDialWarmupSamples = 2_000
	defaultDialP50Budget = 5000 * time.Nanosecond
)

func requireUDSBenchEnv(t *testing.T) (sock, tcpAddr string) {
	t.Helper()
	sock = os.Getenv("REDIS_UDS_SOCKET")
	tcpAddr = os.Getenv("REDIS_TCP_ADDR")
	if sock == "" || tcpAddr == "" {
		t.Skip("REDIS_UDS_SOCKET and REDIS_TCP_ADDR required; run scripts/perf/redis_uds_benchmark.sh")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Skipf("redis uds bench socket unavailable: %v", err)
	}
	_ = conn.Close()
	return sock, tcpAddr
}

func TestRedisUDS_DialLatencyGate(t *testing.T) {
	sock, _ := requireUDSBenchEnv(t)

	latencies := make([]time.Duration, 0, udsDialSamples+udsDialWarmupSamples)
	for range udsDialWarmupSamples + udsDialSamples {
		start := monotonicNano()
		conn, err := shardUniversalOptions(&config.Config{
			RedisAddrs: []string{sock},
		}, 0, nil, RedisShardOptions{}).Dialer(context.Background(), "unix", sock)
		if err != nil {
			t.Fatalf("unix dial failed: %v", err)
		}
		latencies = append(latencies, time.Duration(monotonicNano()-start))
		_ = conn.Close()
	}
	latencies = latencies[udsDialWarmupSamples:]

	p50 := percentileDuration(latencies, 50)
	p99 := percentileDuration(latencies, 99)
	budget := dialP50Budget()
	t.Logf("redis UDS dial n=%d p50=%v p99=%v budget=%v", len(latencies), p50, p99, budget)
	require.Less(t, p50, budget, "UDS dial p50 must stay under budget")
}

func TestRedisUDS_PingLatency(t *testing.T) {
	sock, tcpAddr := requireUDSBenchEnv(t)
	password := os.Getenv("REDIS_PASSWORD")

	udsRDB := redis.NewUniversalClient(shardUniversalOptions(&config.Config{
		RedisPassword: config.Secret(password),
		RedisAddrs:    []string{sock},
	}, 0, nil, RedisShardOptions{}))
	defer func() { _ = udsRDB.Close() }()

	tcpRDB := redis.NewClient(&redis.Options{
		Addr:     tcpAddr,
		Password: password,
	})
	defer func() { _ = tcpRDB.Close() }()

	ctx := context.Background()
	for range 500 {
		require.NoError(t, udsRDB.Ping(ctx).Err())
		require.NoError(t, tcpRDB.Ping(ctx).Err())
	}

	udsLat := samplePingLatency(ctx, t, udsRDB, 2000)
	tcpLat := samplePingLatency(ctx, t, tcpRDB, 2000)

	udsP50 := percentileDuration(udsLat, 50)
	tcpP50 := percentileDuration(tcpLat, 50)
	t.Logf("redis PING uds p50=%v tcp p50=%v", udsP50, tcpP50)
	require.Less(t, udsP50, tcpP50, "UDS PING p50 should beat TCP loopback on same host")
}

func TestRedisUDS_DialVsTCP(t *testing.T) {
	sock, tcpAddr := requireUDSBenchEnv(t)

	uds := sampleDialLatency(t, sock, true, udsDialWarmupSamples+udsDialSamples)[udsDialWarmupSamples:]
	tcp := sampleDialLatency(t, tcpAddr, false, udsDialWarmupSamples+udsDialSamples)[udsDialWarmupSamples:]

	udsP50 := percentileDuration(uds, 50)
	tcpP50 := percentileDuration(tcp, 50)
	t.Logf("transport dial uds p50=%v tcp p50=%v ratio=%.2f", udsP50, tcpP50, float64(tcpP50)/float64(udsP50))
	require.Less(t, udsP50, tcpP50, "UDS dial p50 must beat TCP loopback dial")
}

func sampleDialLatency(t *testing.T, addr string, unixTransport bool, n int) []time.Duration {
	t.Helper()
	out := make([]time.Duration, 0, n)
	for range n {
		start := monotonicNano()
		var conn net.Conn
		var err error
		if unixTransport {
			d := shardUniversalOptions(&config.Config{RedisAddrs: []string{addr}}, 0, nil, RedisShardOptions{}).Dialer
			conn, err = d(context.Background(), "unix", addr)
		} else {
			var d net.Dialer
			conn, err = d.DialContext(context.Background(), "tcp", addr)
		}
		if err != nil {
			t.Fatalf("dial failed: %v", err)
		}
		out = append(out, time.Duration(monotonicNano()-start))
		_ = conn.Close()
	}
	return out
}

func samplePingLatency(ctx context.Context, t *testing.T, rdb redis.UniversalClient, n int) []time.Duration {
	t.Helper()
	out := make([]time.Duration, 0, n)
	for range n {
		start := monotonicNano()
		require.NoError(t, rdb.Ping(ctx).Err())
		out = append(out, time.Duration(monotonicNano()-start))
	}
	return out
}

func percentileDuration(samples []time.Duration, pct int) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	if pct < 1 {
		pct = 1
	}
	if pct > 100 {
		pct = 100
	}
	cp := append([]time.Duration(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := len(cp)*pct/100 - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func dialP50Budget() time.Duration {
	if v := os.Getenv("UDS_DIAL_P50_BUDGET_NS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return time.Duration(n)
		}
	}
	return defaultDialP50Budget
}
