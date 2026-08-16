package fraud

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMicroBatch_AggregationAndScoring(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	scorer, err := NewLGBMScorer(testModelPath(t))
	require.NoError(t, err)

	mb := NewMicroBatcher([]redis.UniversalClient{rdb}, scorer, "")
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
	val, err := rdb.Get(ctx, key).Result()
	require.NoError(t, err)

	assert.NotEmpty(t, val)
	ttl, err := rdb.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= ScoreBoostTTL)
}

func TestMicroBatch_StreamLagPause(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	mb := NewMicroBatcher([]redis.UniversalClient{rdb}, nil, "")

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
}

func TestMicroBatch_BoundedQueueDrop(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	mb := NewMicroBatcher([]redis.UniversalClient{rdb}, nil, "")

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
	go func() {
		mb.Enqueue(evt, msgID)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Enqueue blocked on full channel")
	}
}

func TestFault_MLProcessorLag(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	mb := NewMicroBatcher([]redis.UniversalClient{rdb}, nil, "")

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
