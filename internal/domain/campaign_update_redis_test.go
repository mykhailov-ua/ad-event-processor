package domain

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestPublishCampaignUpdateRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	const channel = "campaigns:update-test"
	campID := uuid.New().String()

	pubsub := rdb.Subscribe(context.Background(), channel)
	defer func() { _ = pubsub.Close() }()

	require.NoError(t, PublishCampaignUpdateRedis(context.Background(), []redis.UniversalClient{rdb}, channel, campID))

	msg, err := pubsub.ReceiveMessage(context.Background())
	require.NoError(t, err)
	require.Equal(t, campID, msg.Payload)

	epoch, err := rdb.Get(context.Background(), CampaignEpochKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1), epoch)
}

func TestDefaultCampaignUpdateChannel(t *testing.T) {
	require.Equal(t, "campaigns:update", DefaultCampaignUpdateChannel(""))
	require.Equal(t, "custom", DefaultCampaignUpdateChannel("custom"))
}
