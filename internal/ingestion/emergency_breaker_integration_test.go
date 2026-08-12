package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmergencyCircuitBreaker_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		RateLimitPerMin:   100,
		RateLimitWindowMs: 60000,
	}

	ctx := context.Background()

	sw := NewSettingsWatcher([]redis.UniversalClient{rdb}, cfg)
	assert.False(t, sw.Get().EmergencyBreaker)

	require.NoError(t, rdb.HSet(ctx, "config:values", "emergency_breaker", "true").Err())

	watcherCtx, cancelWatcher := context.WithCancel(ctx)
	go sw.Start(watcherCtx, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return sw.Get().EmergencyBreaker
	}, 1*time.Second, 20*time.Millisecond)

	cancelWatcher()

	breakerFilter := NewEmergencyBreakerFilter(sw)
	testEvt := &domain.Event{
		CampaignID: uuid.New(),
		Type:       "click",
		ClickID:    "click123",
		UserID:     "user1",
		IP:         "1.1.1.1",
	}

	err := breakerFilter.Check(ctx, testEvt)
	assert.ErrorIs(t, err, domain.ErrEmergencyBreakerActive)

	require.NoError(t, rdb.HSet(ctx, "config:values", "emergency_breaker", "false").Err())

	watcherCtx2, cancelWatcher2 := context.WithCancel(ctx)
	go sw.Start(watcherCtx2, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return !sw.Get().EmergencyBreaker
	}, 1*time.Second, 20*time.Millisecond)

	cancelWatcher2()

	err = breakerFilter.Check(ctx, testEvt)
	assert.NoError(t, err)
}
