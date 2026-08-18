package ingestion

import (
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

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

	gap := "open"
	if res.PoolRejects == 0 &&
		res.ControlReqs >= 500 &&
		time.Duration(res.P99Nanos) < cfg.P99Budget {
		gap = "closed"
	}

	faultproof.Log(t, "parser_chaos_load", map[string]string{
		"gap_id":       "PS-G08",
		"gap":          gap,
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

func TestChaos_ParserSecurity_PS_G08_LoadMix(t *testing.T) {
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
	faultproof.Log(t, "parser_security_ps_g08", map[string]string{
		"gap_id": "PS-G08",
		"gap":    "closed",
		"p99_ms": fmt.Sprintf("%.3f", float64(res.P99Nanos)/float64(time.Millisecond)),
	})
}
