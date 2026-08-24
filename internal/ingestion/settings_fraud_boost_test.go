package ingestion

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyFraudBoostCampaign_updateAndRemove(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sw := NewSettingsWatcher([]redis.UniversalClient{rdb}, &config.Config{})

	ctx := context.Background()
	campID := uuid.New()
	key := fraudScoreBoostKey(campID)

	require.NoError(t, rdb.Set(ctx, key, "42", 0).Err())
	sw.applyFraudBoostCampaign(ctx, campID)
	require.Equal(t, uint8(42), sw.GetFraudScoreBoosts().Boosts[campID])

	require.NoError(t, rdb.Del(ctx, key).Err())
	sw.applyFraudBoostCampaign(ctx, campID)
	_, ok := sw.GetFraudScoreBoosts().Boosts[campID]
	require.False(t, ok)
}

func TestSettingsWatcher_fraudBoostSubscriber(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	const channel = "campaigns:update-test"

	cfg := &config.Config{CampaignUpdateChannel: channel}
	sw := NewSettingsWatcher([]redis.UniversalClient{rdb}, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.runFraudBoostSubscriber(ctx)

	campID := uuid.New()
	key := fraudScoreBoostKey(campID)

	require.NoError(t, rdb.Set(context.Background(), key, "42", 0).Err())
	require.NoError(t, domain.PublishCampaignUpdateRedis(context.Background(), []redis.UniversalClient{rdb}, channel, campID.String()))

	assert.Eventually(t, func() bool {
		return sw.GetFraudScoreBoosts().Boosts[campID] == 42
	}, 2*time.Second, 20*time.Millisecond)
}

func TestParseFraudBoostValue(t *testing.T) {
	score, ok := parseFraudBoostValue("55")
	assert.True(t, ok)
	assert.Equal(t, uint8(55), score)

	score, ok = parseFraudBoostValue("150")
	assert.True(t, ok)
	assert.Equal(t, uint8(100), score)

	_, ok = parseFraudBoostValue("bad")
	assert.False(t, ok)
}
