package ingest

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type chaosLoadConfig struct {
	Duration  time.Duration
	RPS       int
	ChaosPct  int
	P99Budget time.Duration
	Workers   int
}

type chaosLoadResult struct {
	TotalReqs   int64
	ControlReqs int64
	ChaosReqs   int64
	PoolRejects float64
	P99Nanos    int64
	AchievedRPS float64
}

func chaosLoadConfigFromEnv() chaosLoadConfig {
	cfg := chaosLoadConfig{
		Duration:  300 * time.Second,
		RPS:       5000,
		ChaosPct:  10,
		P99Budget: 80 * time.Millisecond,
		Workers:   8,
	}
	if v := os.Getenv("CHAOS_LOAD_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Duration = d
		}
	}
	if v := os.Getenv("CHAOS_LOAD_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RPS = n
		}
	}
	if v := os.Getenv("CHAOS_LOAD_CHAOS_PCT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ChaosPct = n
		}
	}
	if v := os.Getenv("CHAOS_LOAD_P99_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.P99Budget = time.Duration(n) * time.Millisecond
		}
	}
	if v := os.Getenv("CHAOS_LOAD_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Workers = n
		}
	}
	return cfg
}

func chaosProtoTrackBody(seq int) []byte {
	id := uuid.New()
	evt := &pb.AdEvent{
		CampaignId: id[:],
		EventType:  []byte("click"),
		Metadata: &pb.EventMetadata{
			ClickId: []byte("chaos-click"),
			UserId:  []byte(fmt.Sprintf("chaos-%d", seq)),
		},
	}
	body, err := evt.MarshalVT()
	if err != nil {
		panic(err)
	}
	return body
}

func chaosParserWire(seq int) []byte {
	switch seq % 4 {
	case 0:
		return []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n")
	case 1:
		return BuildGnetHTTP("POST", "/track", map[string]string{
			"Content-Type":   "application/x-protobuf",
			"Content-Length": "3",
		}, []byte{0xFF, 0xEE, 0xDD})
	case 2:
		garbage := make([]byte, 0, 128)
		garbage = append(garbage, "POST /openrtb/bid HTTP/1.1\r\nContent-Length: 64\r\n\r\n"...)
		for range 64 {
			garbage = append(garbage, '"')
		}
		return garbage
	default:
		return []byte("GET /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	}
}

func buildChaosControlWire(body []byte) []byte {
	return BuildGnetHTTP("POST", "/track", map[string]string{
		"Content-Type": "application/x-protobuf",
		"Connection":   "keep-alive",
	}, body)
}

func runParserChaosLoad(cfg chaosLoadConfig) chaosLoadResult {
	if cfg.Duration <= 0 {
		cfg.Duration = 2 * time.Second
	}
	if cfg.RPS <= 0 {
		cfg.RPS = 2000
	}
	if cfg.ChaosPct < 0 {
		cfg.ChaosPct = 0
	}
	if cfg.ChaosPct > 100 {
		cfg.ChaosPct = 100
	}
	if cfg.P99Budget <= 0 {
		cfg.P99Budget = 80 * time.Millisecond
	}
	workers := cfg.Workers
	if workers <= 0 {
		workers = 4
	}

	poolRejectsBefore := testutil.ToFloat64(metrics.WorkerPoolRejectTotal)

	pool := NewPinnedWorkerPool(workers, workers*8)
	defer pool.Shutdown()

	handlerCfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(handlerCfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	h.SetWorkerPool(pool)

	interval := time.Second / time.Duration(cfg.RPS)
	if interval <= 0 {
		interval = time.Microsecond
	}

	var (
		totalReqs   atomic.Int64
		controlReqs atomic.Int64
		chaosReqs   atomic.Int64
		controlLats []time.Duration
		latMu       sync.Mutex
	)

	start := time.Now()
	deadline := start.Add(cfg.Duration)
	seq := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		seq++
		isChaos := seq%100 < cfg.ChaosPct
		var inbound []byte
		if isChaos {
			inbound = chaosParserWire(seq)
			chaosReqs.Add(1)
		} else {
			body := chaosProtoTrackBody(seq)
			inbound = buildChaosControlWire(body)
			controlReqs.Add(1)
		}
		t0 := time.Now()
		ServeGnetHarness(h, inbound)
		if !isChaos {
			elapsed := time.Since(t0)
			latMu.Lock()
			controlLats = append(controlLats, elapsed)
			latMu.Unlock()
		}
		totalReqs.Add(1)
	}

	elapsed := time.Since(start)
	p99 := percentileDuration(controlLats, 99)
	achieved := float64(totalReqs.Load()) / elapsed.Seconds()
	if elapsed <= 0 {
		achieved = 0
	}

	return chaosLoadResult{
		TotalReqs:   totalReqs.Load(),
		ControlReqs: controlReqs.Load(),
		ChaosReqs:   chaosReqs.Load(),
		PoolRejects: testutil.ToFloat64(metrics.WorkerPoolRejectTotal) - poolRejectsBefore,
		P99Nanos:    p99.Nanoseconds(),
		AchievedRPS: achieved,
	}
}

func TestChaos_ParserLoad_CX02(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: parser chaos load (run make test-integration)")
	}

	cfg := chaosLoadConfigFromEnv()
	if cfg.Duration > 30*time.Second {
		cfg.Duration = 30 * time.Second
	}
	if cfg.RPS > 8000 {
		cfg.RPS = 8000
	}

	res := runParserChaosLoad(cfg)
	p99Ms := float64(res.P99Nanos) / float64(time.Millisecond)

	proof := "open"
	if res.PoolRejects == 0 &&
		res.ControlReqs >= 500 &&
		time.Duration(res.P99Nanos) < cfg.P99Budget {
		proof = "closed"
	}

	faultproof.Log(t, "parser_chaos_load", map[string]string{
		"case_id":      "parser_chaos_cross_hop",
		"proof":        proof,
		"total_reqs":   fmt.Sprintf("%d", res.TotalReqs),
		"control_reqs": fmt.Sprintf("%d", res.ControlReqs),
		"chaos_reqs":   fmt.Sprintf("%d", res.ChaosReqs),
		"pool_rejects": fmt.Sprintf("%.0f", res.PoolRejects),
		"p99_ms":       fmt.Sprintf("%.3f", p99Ms),
		"achieved_rps": fmt.Sprintf("%.0f", res.AchievedRPS),
		"duration":     cfg.Duration.String(),
		"target_rps":   fmt.Sprintf("%d", cfg.RPS),
		"chaos_pct":    fmt.Sprintf("%d", cfg.ChaosPct),
	})

	require.Equal(t, float64(0), res.PoolRejects, "WorkerPoolRejectTotal must not increase")
	require.GreaterOrEqual(t, res.ControlReqs, int64(500), "need enough control samples")
	require.Less(t, time.Duration(res.P99Nanos), cfg.P99Budget,
		"control cohort p99 %v exceeds budget %v", time.Duration(res.P99Nanos), cfg.P99Budget)
}

func TestChaos_ParserSecurity_LoadMix(t *testing.T) {
	cfg := chaosLoadConfig{
		Duration:  2 * time.Second,
		RPS:       2000,
		ChaosPct:  10,
		P99Budget: 80 * time.Millisecond,
		Workers:   4,
	}
	res := runParserChaosLoad(cfg)
	require.Equal(t, float64(0), res.PoolRejects)
	require.Greater(t, res.ControlReqs, int64(100))
	faultproof.Log(t, "parser_security_parser_chaos_cross_hop", map[string]string{
		"case_id": "parser_chaos_cross_hop",
		"proof":   "closed",
		"p99_ms":  fmt.Sprintf("%.3f", float64(res.P99Nanos)/float64(time.Millisecond)),
	})
}
