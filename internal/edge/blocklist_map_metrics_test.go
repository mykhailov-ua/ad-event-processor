package edge

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ad-event-processor/internal/metrics"
)

func TestRecordBlocklistChangelogLagSeconds(t *testing.T) {
	ctx := context.Background()
	now := time.Now().Unix()
	stub := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	state := &BlocklistSyncState{lastScore: float64(now - 120)}

	RecordBlocklistChangelogLagSeconds(ctx, stub, state)
	lag := testutil.ToFloat64(metrics.EdgeBlocklistChangelogLagSeconds)
	assert.GreaterOrEqual(t, lag, 119.0)
	assert.LessOrEqual(t, lag, 125.0)
}

func TestRecordBlocklistMapMetrics_nilMaps(t *testing.T) {
	require.NotPanics(t, func() {
		RecordBlocklistMapMetrics(BlocklistMaps{}, nil)
		RecordBlocklistMapMetrics(BlocklistMaps{}, NewBlocklistStore())
	})
}
