package outbox_test

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/edge"
	"ad-event-processor/internal/outbox"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyBlacklistPayload_publishesQuarantine(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	redisClient, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	pubsub := redisClient.Subscribe(ctx, edge.FraudQuarantineChannel)
	defer pubsub.Close()
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	worker := newRedisWorker([]redis.UniversalClient{redisClient})

	ip := "203.0.113.10"
	require.NoError(t, worker.ApplyBlacklistPayload(ctx, outbox.BlacklistPayload{
		Action: "add",
		IP:     ip,
		Reason: "fraud",
	}, time.Now()))

	msg, err := pubsub.ReceiveTimeout(ctx, 3*time.Second)
	require.NoError(t, err)
	payload, ok := msg.(*redis.Message)
	require.True(t, ok)
	assert.Equal(t, edge.FraudQuarantineChannel, payload.Channel)
	expected, err := edge.MarshalFraudQuarantinePayload([]string{ip})
	require.NoError(t, err)
	assert.Equal(t, expected, payload.Payload)

	isMember, err := redisClient.SIsMember(ctx, "blacklist:fraud", ip).Result()
	require.NoError(t, err)
	assert.True(t, isMember)
}
