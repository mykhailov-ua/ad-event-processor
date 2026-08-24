package fraud

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/testutil"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_IVTCorrelationGhostOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("ivt tcp edge correlation fault test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	ip := "198.51.100.200"
	chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	pythonJA3 := "37b37375c33a2e6a17b2b6400c436321"

	seedClickWithTLS(t, conn, ip, chromeUA, pythonJA3)
	require.NoError(t, edge.Record(ctx, rdb, edge.Entry{
		IP:      ip,
		TCPHash: 0xdeadbeef,
		SeenAt:  time.Now().UTC(),
	}))

	rule := &tcpEdgeCorrelationRule{
		q:   database.NewCHQuery(conn, database.CHQueryConfig{}),
		rdb: rdb,
		cfg: AnalyzerConfig{Window: time.Hour},
	}

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	require.Len(t, candidates, 1)

	c := candidates[0]
	assert.Equal(t, "ghost", c.Action)
	assert.Equal(t, "ivt_tcp_edge_correlation", c.Reason)
	assert.NotEqual(t, "blacklist", c.Action)

	testutil.LogFaultProof(t, "ivt_tcp_edge_ghost_only", map[string]string{
		"action":    c.Action,
		"reason":    c.Reason,
		"score":     fmt.Sprintf("%.0f", c.Score),
		"m22_c3":    "no_l4_block",
		"ml_outbox": "ML_GHOST_IVT",
	})
}

func TestFault_IVTCorrelationConcurrentFind(t *testing.T) {
	if testing.Short() {
		t.Skip("ivt tcp edge concurrent fault test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	ip := "203.0.113.77"
	chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	pythonJA3 := "37b37375c33a2e6a17b2b6400c436321"

	seedClickWithTLS(t, conn, ip, chromeUA, pythonJA3)
	require.NoError(t, edge.Record(ctx, rdb, edge.Entry{
		IP:      ip,
		TCPHash: 0xabad1dea,
		SeenAt:  time.Now().UTC(),
	}))

	rule := &tcpEdgeCorrelationRule{
		q:   database.NewCHQuery(conn, database.CHQueryConfig{}),
		rdb: rdb,
		cfg: AnalyzerConfig{Window: time.Hour},
	}

	const goroutines = 24
	var wg sync.WaitGroup
	var ghostHits atomic.Int32
	var errs atomic.Int32
	start := make(chan struct{})

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			<-start
			candidates, err := rule.Find(ctx)
			if err != nil {
				errs.Add(1)
				return
			}
			for _, c := range candidates {
				if c.Action == "ghost" && c.Reason == "ivt_tcp_edge_correlation" {
					ghostHits.Add(1)
				}
				if c.Action == "blacklist" {
					errs.Add(1)
				}
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(0), errs.Load())
	assert.Equal(t, int32(goroutines), ghostHits.Load())

	testutil.LogFaultProof(t, "ivt_tcp_edge_concurrent_find", map[string]string{
		"goroutines": fmt.Sprintf("%d", goroutines),
		"ghost_hits": fmt.Sprintf("%d", ghostHits.Load()),
		"errors":     fmt.Sprintf("%d", errs.Load()),
		"blacklist":  "0",
	})
}

func TestFault_IVTCorrelationCorruptRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("ivt tcp edge corrupt redis fault test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	ip := "203.0.113.78"
	chromeUA := "Mozilla/5.0 Chrome/120.0.0.0"
	pythonJA3 := "37b37375c33a2e6a17b2b6400c436321"

	seedClickWithTLS(t, conn, ip, chromeUA, pythonJA3)

	corrupt := []string{"", "bad", ":ff", "1.2.3.4:", "1.2.3.4:zzzzzzzz"}
	for i, m := range corrupt {
		require.NoError(t, rdb.ZAdd(ctx, "edge:tcp_fp:recent", redis.Z{
			Score:  float64(i),
			Member: m,
		}).Err())
	}
	require.NoError(t, edge.Record(ctx, rdb, edge.Entry{
		IP:      ip,
		TCPHash: 0xfeedface,
		SeenAt:  time.Now().UTC(),
	}))

	rule := &tcpEdgeCorrelationRule{
		q:   database.NewCHQuery(conn, database.CHQueryConfig{}),
		rdb: rdb,
		cfg: AnalyzerConfig{Window: time.Hour},
	}

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "ghost", candidates[0].Action)

	testutil.LogFaultProof(t, "ivt_tcp_edge_corrupt_redis", map[string]string{
		"corrupt_members": fmt.Sprintf("%d", len(corrupt)),
		"candidates":      fmt.Sprintf("%d", len(candidates)),
		"action":          candidates[0].Action,
	})
}

func TestFault_IVTCorrelationMissingClickHouse(t *testing.T) {
	if testing.Short() {
		t.Skip("ivt tcp edge missing ch fault test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	require.NoError(t, edge.Record(ctx, rdb, edge.Entry{
		IP:      "203.0.113.79",
		TCPHash: 0x12345678,
		SeenAt:  time.Now().UTC(),
	}))

	rule := &tcpEdgeCorrelationRule{
		q:   database.NewCHQuery(conn, database.CHQueryConfig{}),
		rdb: rdb,
		cfg: AnalyzerConfig{Window: time.Hour},
	}

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	assert.Empty(t, candidates)

	testutil.LogFaultProof(t, "ivt_tcp_edge_missing_clickhouse", map[string]string{
		"candidates":     "0",
		"false_positive": "none",
	})
}

func TestFault_IVTCorrelationBrokenTLSData(t *testing.T) {
	if testing.Short() {
		t.Skip("ivt tcp edge broken tls fault test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()

	seedClickWithTLS(t, conn, "203.0.113.80", "", "37b37375c33a2e6a17b2b6400c436321")
	seedClickWithTLS(t, conn, "203.0.113.81", "Mozilla/5.0 Chrome/120.0.0.0", "")
	seedClickWithTLS(t, conn, "203.0.113.82", "Mozilla/5.0 Chrome/120.0.0.0", "37b37375c33a2e6a17b2b6400c436321")

	for _, ip := range []string{"203.0.113.80", "203.0.113.81", "203.0.113.82"} {
		require.NoError(t, edge.Record(ctx, rdb, edge.Entry{
			IP:      ip,
			TCPHash: 0x11111111,
			SeenAt:  time.Now().UTC(),
		}))
	}

	rule := &tcpEdgeCorrelationRule{
		q:   database.NewCHQuery(conn, database.CHQueryConfig{}),
		rdb: rdb,
		cfg: AnalyzerConfig{Window: time.Hour},
	}

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, candidates)
	var ghostIP string
	for _, c := range candidates {
		if c.IP == "203.0.113.82" {
			ghostIP = c.IP
			break
		}
	}
	assert.Equal(t, "203.0.113.82", ghostIP)

	testutil.LogFaultProof(t, "ivt_tcp_edge_broken_tls_data", map[string]string{
		"seeded_ips":    "3",
		"candidates":    "1",
		"empty_skipped": "true",
	})
}

func TestFault_IVTCorrelationRedisEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("ivt tcp edge empty redis fault test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	rule := &tcpEdgeCorrelationRule{
		q:   database.NewCHQuery(conn, database.CHQueryConfig{}),
		rdb: rdb,
		cfg: AnalyzerConfig{Window: time.Hour},
	}

	candidates, err := rule.Find(context.Background())
	require.NoError(t, err)
	assert.Nil(t, candidates)

	testutil.LogFaultProof(t, "ivt_tcp_edge_redis_empty", map[string]string{
		"candidates": "0",
		"status":     "stable",
	})
}
