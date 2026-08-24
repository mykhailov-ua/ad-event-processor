package controlplane

import (
	"context"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_ControlRedisTerminateOutboxStaysPending(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupControlFaultInfra(t)
	defer cleanup()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:control-redis-kill"}
	svc := newBareService(t, infra.Pool, []redis.UniversalClient{infra.Redis}, cfg)
	ctx := context.Background()

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"rate_limit_per_min": "77"}))
	eventID := latestOutboxEventID(t, infra.Pool, "UPDATE_SETTINGS")

	worker := NewOutboxWorker(svc)
	require.NoError(t, infra.RedisContainer.Terminate(ctx))
	requireFaultActive(t, func() bool {
		return infra.Redis.Ping(ctx).Err() != nil
	}, "redis ping must fail after terminate")

	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.Error(t, err)
	require.Equal(t, 0, processed)

	status := outboxStatus(t, infra.Pool, eventID)
	require.Equal(t, "PENDING", status)

	faultproof.Log(t, "redis_container_terminate", map[string]string{
		"subsystem":    "management_outbox",
		"op":           "process_outbox",
		"baseline_ok":  "true",
		"status":       status,
		"processed":    "0",
		"fault_verify": "redis_container_terminated",
	})
}

func TestFault_ControlRedisStopStartOutboxRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupControlFaultInfra(t)
	defer cleanup()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:control-redis-recovery"}
	svc := newBareService(t, infra.Pool, []redis.UniversalClient{infra.Redis}, cfg)
	ctx := context.Background()

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"rate_limit_per_min": "88"}))
	eventID := latestOutboxEventID(t, infra.Pool, "UPDATE_SETTINGS")

	worker := NewOutboxWorker(svc)

	stopFaultContainer(t, infra.RedisContainer)
	requireFaultActive(t, func() bool {
		return infra.Redis.Ping(ctx).Err() != nil
	}, "redis ping must fail after stop")

	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.Error(t, err)
	require.Equal(t, 0, processed)
	require.Equal(t, "PENDING", outboxStatus(t, infra.Pool, eventID))

	startFaultContainer(t, infra.RedisContainer)
	infra.refreshRedisClient(t)
	rebindBareService(svc, infra)

	recovered := false
	require.Eventually(t, func() bool {
		n, err := worker.ProcessOutboxWithCount(ctx, 10)
		if err != nil || n != 1 {
			return false
		}
		recovered = outboxStatus(t, infra.Pool, eventID) == "PROCESSED"
		return recovered
	}, 30*time.Second, 200*time.Millisecond, "outbox must drain after Redis restart")

	val, err := infra.Redis.HGet(ctx, "config:values", "rate_limit_per_min").Result()
	require.NoError(t, err)
	require.Equal(t, "88", val)

	faultproof.Log(t, "redis_stop_start_recovery", map[string]string{
		"subsystem":    "management_outbox",
		"op":           "process_outbox",
		"baseline_ok":  "true",
		"recovered":    strconv.FormatBool(recovered),
		"fault_verify": "redis_container_stopped_then_started",
	})
}

func TestFault_ControlPGStopOutboxClaimBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupControlFaultInfra(t)
	defer cleanup()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:control-pg-stop"}
	svc := newBareService(t, infra.Pool, []redis.UniversalClient{infra.Redis}, cfg)
	ctx := context.Background()

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"emergency_breaker": "false"}))
	eventID := latestOutboxEventID(t, infra.Pool, "UPDATE_SETTINGS")

	worker := NewOutboxWorker(svc)

	stopFaultContainer(t, infra.PGContainer)
	requireFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after stop")

	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.Error(t, err)
	require.Equal(t, 0, processed)

	startFaultContainer(t, infra.PGContainer)
	infra.refreshPGPool(t)
	rebindBareService(svc, infra)
	require.Equal(t, "PENDING", outboxStatus(t, infra.Pool, eventID))

	faultproof.Log(t, "postgres_container_stop", map[string]string{
		"subsystem":    "management_outbox",
		"op":           "process_outbox",
		"baseline_ok":  "true",
		"status":       "PENDING",
		"processed":    "0",
		"fault_verify": "postgres_container_stopped",
	})
}

func TestFault_ControlPGStopStartOutboxRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupControlFaultInfra(t)
	defer cleanup()

	cfg := &config.Config{CampaignUpdateChannel: "campaigns:control-pg-recovery"}
	svc := newBareService(t, infra.Pool, []redis.UniversalClient{infra.Redis}, cfg)
	ctx := context.Background()

	require.NoError(t, svc.UpdateSettings(ctx, map[string]string{"emergency_breaker": "true"}))
	eventID := latestOutboxEventID(t, infra.Pool, "UPDATE_SETTINGS")

	worker := NewOutboxWorker(svc)

	stopFaultContainer(t, infra.PGContainer)
	requireFaultActive(t, func() bool {
		return infra.Pool.Ping(ctx) != nil
	}, "pg ping must fail after stop")

	startFaultContainer(t, infra.PGContainer)
	infra.refreshPGPool(t)
	rebindBareService(svc, infra)

	recovered := false
	require.Eventually(t, func() bool {
		n, err := worker.ProcessOutboxWithCount(ctx, 10)
		if err != nil || n != 1 {
			return false
		}
		recovered = outboxStatus(t, infra.Pool, eventID) == "PROCESSED"
		return recovered
	}, 30*time.Second, 200*time.Millisecond, "outbox must drain after Postgres restart")

	val, err := infra.Redis.HGet(ctx, "config:values", "emergency_breaker").Result()
	require.NoError(t, err)
	require.Equal(t, "true", val)

	faultproof.Log(t, "postgres_stop_start_recovery", map[string]string{
		"subsystem":    "management_outbox",
		"op":           "process_outbox",
		"baseline_ok":  "true",
		"recovered":    strconv.FormatBool(recovered),
		"fault_verify": "postgres_container_stopped_then_started",
	})
}
