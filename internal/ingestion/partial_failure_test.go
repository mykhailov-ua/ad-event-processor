package ingestion

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type mockPinger struct {
	fail bool
}

func (m *mockPinger) Ping(ctx context.Context) error {
	if m.fail {
		return errors.New("ping failed")
	}
	return nil
}

type mockFailRedis struct {
	redis.UniversalClient
	fail bool
}

func (m *mockFailRedis) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.fail {
		cmd.SetErr(errors.New("redis connection refused"))
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func TestHealthCheckPartialFailure(t *testing.T) {
	cfg := &config.Config{NodeWarmupSec: 300}
	registry := &mockRegistry{}

	t.Run("All Healthy", func(t *testing.T) {
		redisShards := []redis.UniversalClient{&mockFailRedis{fail: false}}
		pool := &mockPinger{fail: false}
		sharder := NewJumpHashSharder(1)
		handler := NewAdsPacketHandler(cfg, registry, nil, pool, redisShards, sharder, "fraud-stream", nil)
		handler.SetHealthProbeState(true, true)
		handler.SetStartedAt(time.Now().Add(-301 * time.Second))

		status, body := GetHealthGnet(handler)
		assert.Equal(t, http.StatusOK, status)
		assert.Contains(t, body, "OK")

		status, body = GetReadyGnet(handler)
		assert.Equal(t, http.StatusOK, status)
		assert.Contains(t, body, "OK")
	})

	t.Run("Postgres Down", func(t *testing.T) {
		redisShards := []redis.UniversalClient{&mockFailRedis{fail: false}}
		pool := &mockPinger{fail: true}
		sharder := NewJumpHashSharder(1)
		handler := NewAdsPacketHandler(cfg, registry, nil, pool, redisShards, sharder, "fraud-stream", nil)
		handler.SetHealthProbeState(false, true)
		handler.SetStartedAt(time.Now().Add(-301 * time.Second))

		status, body := GetHealthGnet(handler)
		assert.Equal(t, http.StatusOK, status, "/health is liveness-only")
		assert.Contains(t, body, "OK")

		status, body = GetReadyGnet(handler)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.Contains(t, body, "not ready")
	})

	t.Run("Redis Shard 2 Down", func(t *testing.T) {
		redisShards := []redis.UniversalClient{
			&mockFailRedis{fail: false},
			&mockFailRedis{fail: true},
		}
		pool := &mockPinger{fail: false}
		sharder := NewJumpHashSharder(1)
		handler := NewAdsPacketHandler(cfg, registry, nil, pool, redisShards, sharder, "fraud-stream", nil)
		handler.SetHealthProbeState(false, true, false)
		handler.SetStartedAt(time.Now().Add(-301 * time.Second))

		status, body := GetHealthGnet(handler)
		assert.Equal(t, http.StatusOK, status, "/health is liveness-only")
		assert.Contains(t, body, "OK")

		status, body = GetReadyGnet(handler)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.Contains(t, body, "not ready")
	})
}
