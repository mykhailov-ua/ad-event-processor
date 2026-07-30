package ingestion

import (
	"context"
	"testing"
	"time"

	"espx/internal/database"
	"espx/internal/metrics"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

const faultRedisShardLabel = "0"

func faultGaugeValue(t *testing.T, g interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, g.Write(&m))
	return m.GetGauge().GetValue()
}

func trackerHealthDegradedMetric(t *testing.T) float64 {
	t.Helper()
	return faultGaugeValue(t, metrics.TrackerHealthDegraded)
}

func redisBreakerStateMetric(t *testing.T, shard string) float64 {
	t.Helper()
	g, err := metrics.RedisBreakerState.GetMetricWithLabelValues(shard)
	require.NoError(t, err)
	return faultGaugeValue(t, g)
}

func requireRedisOutageMetrics(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool {
		if trackerHealthDegradedMetric(t) == 1 {
			return true
		}
		return redisBreakerStateMetric(t, faultRedisShardLabel) == float64(database.CircuitOpen)
	}, 15*time.Second, 200*time.Millisecond,
		"during Redis outage expect ad_tracker_health_degraded=1 or ad_redis_breaker_state{shard=%q}=open",
		faultRedisShardLabel)
}

func requireRedisSteadyStateMetrics(t *testing.T) {
	t.Helper()
	require.Eventually(t, func() bool {
		if trackerHealthDegradedMetric(t) != 0 {
			return false
		}
		return redisBreakerStateMetric(t, faultRedisShardLabel) == float64(database.CircuitClosed)
	}, 30*time.Second, 200*time.Millisecond,
		"steady-state restoration: ad_tracker_health_degraded=0 and ad_redis_breaker_state{shard=%q}=closed",
		faultRedisShardLabel)
}

func tripFaultRedisBreaker(t *testing.T, infra *adsFaultInfra) {
	t.Helper()
	require.NotNil(t, infra.RedisBreaker)
	ctx := context.Background()
	require.Eventually(t, func() bool {
		for range 3 {
			_ = infra.Redis.Ping(ctx).Err()
		}
		return infra.RedisBreaker.State() == database.CircuitOpen
	}, 10*time.Second, 50*time.Millisecond)
}
