package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/edge"

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

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	pubsub := rdb.Subscribe(ctx, edge.FraudQuarantineChannel)
	defer pubsub.Close()
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	svc := &Service{rdbs: []redis.UniversalClient{rdb}}
	worker := &OutboxWorker{svc: svc}

	ip := "203.0.113.10"
	require.NoError(t, worker.applyBlacklistPayload(ctx, BlacklistPayload{
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

	isMember, err := rdb.SIsMember(ctx, "blacklist:fraud", ip).Result()
	require.NoError(t, err)
	assert.True(t, isMember)
}
