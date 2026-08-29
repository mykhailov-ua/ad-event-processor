package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/opsadmin"

	"ad-event-processor/internal/config"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/ingest/pb"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestListDLQInbox_includesStreamEntries(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	campaignID := uuid.New()
	pbDLQ := &pb.AdDLQEvent{
		Error:        []byte("timeout"),
		FailedAtUnix: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC).Unix(),
		RetryCount:   2,
		OriginalEvent: &pb.AdStreamEvent{
			CampaignId: campaignID[:],
			EventType:  []byte("click"),
		},
	}
	raw, err := proto.Marshal(pbDLQ)
	require.NoError(t, err)

	dlqStream := "ad:events:dlq"
	msgID, err := redisClient.XAdd(context.Background(), &redis.XAddArgs{
		Stream: dlqStream,
		Values: map[string]interface{}{"d": ingestion.UnsafeString(raw)},
	}).Result()
	require.NoError(t, err)

	cfg := &config.Config{RedisStreamName: "ad:events:stream"}
	svc := &Service{redisShards: []redis.UniversalClient{redisClient}, cfg: cfg}
	reader := newOpsReader(svc)

	result, err := reader.ListDLQInbox(context.Background(), "stream", "", 50)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "stream", result.Items[0].Source)
	assert.Equal(t, opsadmin.DLQRouteID(0, msgID), result.Items[0].ID)
	assert.Equal(t, campaignID.String(), result.Items[0].CampaignID)
}

func TestDlqInboxSourceFromProvider(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "capi", opsadmin.DLQInboxSourceFromProvider("facebook"))
	assert.Equal(t, "capi", opsadmin.DLQInboxSourceFromProvider("Google"))
	assert.Equal(t, "postback", opsadmin.DLQInboxSourceFromProvider("webhook"))
}

func TestParseInboxStreamRouteID(t *testing.T) {
	t.Parallel()
	id := "shard-2-1700000000000-0"
	assert.Equal(t, 2, opsadmin.ParseInboxStreamShard(id))
	assert.Equal(t, "1700000000000-0", opsadmin.ParseInboxStreamEntryID(id))
}
