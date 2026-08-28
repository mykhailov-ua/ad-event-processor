package opsadmin

import (
	"encoding/json"
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestComputeStackHealthStatus_degradedWhenOutboxOver30s(t *testing.T) {
	snap := StackHealthSnapshot{
		LicenseState:               string(licensing.StateActive),
		OutboxOldestPendingSeconds: 45,
		RedisShardsTotal:           1,
		RedisShardsReachable:       1,
		RedisShardReachable:        true,
		ClickHouseLagSeconds:       1,
	}
	require.Equal(t, "degraded", ComputeStackHealthStatus(snap))
}

func TestComputeStackHealthStatus_criticalWhenOutboxOver300s(t *testing.T) {
	snap := StackHealthSnapshot{
		LicenseState:               string(licensing.StateActive),
		OutboxOldestPendingSeconds: 400,
		RedisShardsTotal:           1,
		RedisShardsReachable:       1,
	}
	require.Equal(t, "critical", ComputeStackHealthStatus(snap))
}

func TestComputeStackHealthStatus_okWhenHealthy(t *testing.T) {
	snap := StackHealthSnapshot{
		LicenseState:               string(licensing.StateActive),
		OutboxOldestPendingSeconds: 5,
		ClickHouseLagSeconds:       10,
		RedisShardsTotal:           2,
		RedisShardsReachable:       2,
		RedisShardReachable:        true,
	}
	require.Equal(t, "ok", ComputeStackHealthStatus(snap))
}

func TestStackHealthSnapshot_marshaledJSONHasNoSecrets(t *testing.T) {
	costSyncAge := 120.0
	automationAge := 30.0
	snap := StackHealthSnapshot{
		Status:                          "degraded",
		ClickHouseLagSeconds:            12.5,
		OutboxOldestPendingSeconds:      45,
		RedisShardReachable:             true,
		RedisShardsReachable:            2,
		RedisShardsTotal:                2,
		CostSyncLastSuccessSeconds:      &costSyncAge,
		AutomationWorkerLastTickSeconds: &automationAge,
		LicenseState:                    string(licensing.StateActive),
	}
	raw, err := json.Marshal(snap)
	require.NoError(t, err)
	require.False(t, StackHealthSnapshotHasSecretMaterial(string(raw)))
}
