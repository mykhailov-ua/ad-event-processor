package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestListDLQEntries_readsRedisShard(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

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
	msgID, err := rdb.XAdd(context.Background(), &redis.XAddArgs{
		Stream: dlqStream,
		Values: map[string]interface{}{"d": ingestion.UnsafeString(raw)},
	}).Result()
	require.NoError(t, err)

	cfg := &config.Config{RedisStreamName: "ad:events:stream"}
	svc := &Service{rdbs: []redis.UniversalClient{rdb}, cfg: cfg}
	reader := newOpsReader(svc)

	result, err := reader.ListDLQEntries(context.Background(), "", 50)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, dlqRouteID(0, msgID), result.Items[0].ID)
	assert.Equal(t, 0, result.Items[0].ShardID)
	assert.Equal(t, campaignID.String(), result.Items[0].CampaignID)
	assert.Equal(t, "click", result.Items[0].EventType)
	assert.Equal(t, "timeout", result.Items[0].Error)
}

func TestEnqueueDLQRetry_requeuesAndDeletes(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	campID := uuid.New()
	event := &pb.AdStreamEvent{
		ClickId:    []byte("clk-1"),
		CampaignId: campID[:],
		EventType:  []byte("impression"),
	}

	dlqStream := "ad:events:dlq"
	targetStream := "ad:events:stream"
	pbDLQ := &pb.AdDLQEvent{
		Error:         []byte("flush failed"),
		FailedAtUnix:  time.Now().Unix(),
		OriginalEvent: event,
	}
	dlqRaw, err := proto.Marshal(pbDLQ)
	require.NoError(t, err)

	msgID, err := rdb.XAdd(context.Background(), &redis.XAddArgs{
		Stream: dlqStream,
		Values: map[string]interface{}{"d": ingestion.UnsafeString(dlqRaw)},
	}).Result()
	require.NoError(t, err)

	cfg := &config.Config{RedisStreamName: targetStream}
	svc := &Service{rdbs: []redis.UniversalClient{rdb}, cfg: cfg}
	reader := newOpsReader(svc)

	err = reader.EnqueueDLQRetry(context.Background(), DLQRetryPayload{
		ShardID: 0,
		EntryID: msgID,
		DLQID:   dlqRouteID(0, msgID),
	}, "idem-1")
	require.NoError(t, err)

	dlqLen, err := rdb.XLen(context.Background(), dlqStream).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), dlqLen)

	targetMsgs, err := rdb.XRange(context.Background(), targetStream, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, targetMsgs, 1)
	assert.Contains(t, targetMsgs[0].Values, "d")
}

func TestEnqueueDLQRetry_idempotent(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	event := &pb.AdStreamEvent{EventType: []byte("click")}
	pbDLQ := &pb.AdDLQEvent{OriginalEvent: event}
	dlqRaw, err := proto.Marshal(pbDLQ)
	require.NoError(t, err)

	msgID, err := rdb.XAdd(context.Background(), &redis.XAddArgs{
		Stream: "ad:events:dlq",
		Values: map[string]interface{}{"d": ingestion.UnsafeString(dlqRaw)},
	}).Result()
	require.NoError(t, err)

	cfg := &config.Config{RedisStreamName: "ad:events:stream"}
	svc := &Service{rdbs: []redis.UniversalClient{rdb}, cfg: cfg}
	reader := newOpsReader(svc)
	payload := DLQRetryPayload{
		ShardID: 0,
		EntryID: msgID,
		DLQID:   dlqRouteID(0, msgID),
	}

	require.NoError(t, reader.EnqueueDLQRetry(context.Background(), payload, "idem-dup"))
	require.NoError(t, reader.EnqueueDLQRetry(context.Background(), payload, "idem-dup"))

	targetLen, err := rdb.XLen(context.Background(), "ad:events:stream").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), targetLen)
}
