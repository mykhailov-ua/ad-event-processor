package scorer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMicroBatch_AggregationAndScoring(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	scorer, err := NewLGBMScorer(testModelPath(t))
	require.NoError(t, err)

	mb := NewMicroBatcher([]redis.UniversalClient{redisClient}, scorer, "", DefaultMicroBatcherConfig())
	go mb.Start(ctx)

	campaignID := uuid.New()

	now := time.Now()
	for range 10 {
		evt := &domain.Event{
			IP:         "1.2.3.4",
			CampaignID: campaignID,
			Type:       "click",
			UserID:     "user1",
			UA:         "ua1",
			CreatedAt:  now,
		}
		msgID := fmt.Sprintf("%d-0", now.UnixNano()/1e6)
		mb.Enqueue(evt, msgID)
	}

	time.Sleep(250 * time.Millisecond)

	key := fmt.Sprintf("ml:score:boost:%s", campaignID.String())
	val, err := redisClient.Get(ctx, key).Result()
	require.NoError(t, err)

	assert.NotEmpty(t, val)
	ttl, err := redisClient.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= ScoreBoostTTL)
}

func TestMicroBatch_StreamLagPause(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()

	mb := NewMicroBatcher([]redis.UniversalClient{redisClient}, nil, "", DefaultMicroBatcherConfig())

	campaignID := uuid.New()
	evt := &domain.Event{
		IP:         "1.2.3.4",
		CampaignID: campaignID,
		Type:       "click",
		CreatedAt:  time.Now(),
	}

	staleTime := time.Now().Add(-40 * time.Second)
	msgID := fmt.Sprintf("%d-0", staleTime.UnixNano()/1e6)

	mb.Enqueue(evt, msgID)

	assert.Len(t, mb.eventsChan, 0)
	assert.True(t, mb.paused.Load())
}

func TestMicroBatch_refreshBoostTTLWhenPaused(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: microbatch redis ttl refresh")
	}

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	mb := NewMicroBatcher([]redis.UniversalClient{redisClient}, nil, "", DefaultMicroBatcherConfig())
	campaignID := uuid.New()
	mb.lastBoosts.Store(map[string]int{campaignID.String(): 42})
	mb.paused.Store(true)

	mb.refreshBoostTTL(ctx)

	key := fmt.Sprintf("ml:score:boost:%s", campaignID.String())
	val, err := redisClient.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, "42", val)
	ttl, err := redisClient.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= ScoreBoostTTL)
}

func TestMicroBatch_BoundedQueueDrop(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()

	mb := NewMicroBatcher([]redis.UniversalClient{redisClient}, nil, "", DefaultMicroBatcherConfig())

	campaignID := uuid.New()
	for range 10000 {
		evt := &domain.Event{
			IP:         "1.2.3.4",
			CampaignID: campaignID,
			Type:       "click",
			CreatedAt:  time.Now(),
		}
		msgID := fmt.Sprintf("%d-0", time.Now().UnixNano()/1e6)
		mb.Enqueue(evt, msgID)
	}

	assert.Len(t, mb.eventsChan, 10000)

	evt := &domain.Event{
		IP:         "1.2.3.4",
		CampaignID: campaignID,
		Type:       "click",
		CreatedAt:  time.Now(),
	}
	msgID := fmt.Sprintf("%d-0", time.Now().UnixNano()/1e6)

	done := make(chan bool, 1)
	beforeDrop := testutil.ToFloat64(metrics.MicroBatchDroppedTotal)
	go func() {
		mb.Enqueue(evt, msgID)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Enqueue blocked on full channel")
	}
	assert.Equal(t, beforeDrop+1, testutil.ToFloat64(metrics.MicroBatchDroppedTotal))
}

func TestFault_MLProcessorLag(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()

	mb := NewMicroBatcher([]redis.UniversalClient{redisClient}, nil, "", DefaultMicroBatcherConfig())

	campaignID := uuid.New()
	evt := &domain.Event{
		IP:         "1.2.3.4",
		CampaignID: campaignID,
		Type:       "click",
		CreatedAt:  time.Now(),
	}

	msgIDNormal := fmt.Sprintf("%d-0", time.Now().UnixNano()/1e6)
	mb.Enqueue(evt, msgIDNormal)
	assert.Len(t, mb.eventsChan, 1)

	staleTime := time.Now().Add(-40 * time.Second)
	msgIDStale := fmt.Sprintf("%d-0", staleTime.UnixNano()/1e6)
	mb.Enqueue(evt, msgIDStale)

	assert.Len(t, mb.eventsChan, 1)

	faultproof.Log(t, "ml_processor_lag", map[string]string{
		"subsystem": "fraud_scoring",
		"lag_sec":   "40.0",
		"paused":    "true",
		"status":    "success",
	})
}
