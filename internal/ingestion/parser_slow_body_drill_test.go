package ingestion

import (
	"fmt"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestParserSlowBodyDrill_P99Isolation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: parser slow-body drill (run make test-integration)")
	}

	cfg := slowBodyDrillConfigFromEnv()
	if cfg.Duration > 12*time.Second {
		cfg.Duration = 12 * time.Second
	}
	if cfg.Connections > 64 {
		cfg.Connections = 64
	}

	res := runParserSlowBodyDrill(cfg)
	p99 := time.Duration(res.P99Nanos)

	gap := "open"
	if res.ControlReqs >= 200 && p99 < cfg.P99Budget {
		gap = "closed"
	}

	faultproof.Log(t, "parser_slow_body_drill", map[string]string{
		"gap_id":       "http1_incomplete_body_spin_close",
		"gap":          gap,
		"connections":  fmt.Sprintf("%d", cfg.Connections),
		"control_reqs": fmt.Sprintf("%d", res.ControlReqs),
		"p99_ms":       fmt.Sprintf("%.3f", float64(p99)/float64(time.Millisecond)),
		"duration":     cfg.Duration.String(),
	})

	require.GreaterOrEqual(t, res.ControlReqs, int64(200), "need control cohort samples")
	require.Less(t, p99, cfg.P99Budget,
		"control p99 %v exceeds budget %v with %d slow drips", p99, cfg.P99Budget, cfg.Connections)
}
