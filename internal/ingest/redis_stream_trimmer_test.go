package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRedisUsedMemory(t *testing.T) {
	sampleInfo := `# Memory
used_memory:10485760
used_memory_human:10.00M
used_memory_rss:12582912`

	bytes := parseRedisUsedMemory(sampleInfo)
	assert.Equal(t, int64(10485760), bytes)

	assert.Equal(t, int64(-1), parseRedisUsedMemory("invalid_info_data"))
}

func TestRedisStreamTrimmer_TrimOnceAndMetrics(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

	ctx := context.Background()
	stream := "test:trim:stream"

	for range 50 {
		_, err := redisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]interface{}{"k": "v"},
		}).Result()
		require.NoError(t, err)
	}

	assert.Equal(t, int64(50), redisClient.XLen(ctx, stream).Val())

	trimmer := NewRedisStreamTrimmer(RedisStreamTrimmerConfig{
		RedisShards:  []redis.UniversalClient{redisClient},
		Streams:      []string{stream},
		MaxLen:       10,
		TrimInterval: 50 * time.Millisecond,
	})

	trimmer.TrimOnce(ctx)

	assert.LessOrEqual(t, redisClient.XLen(ctx, stream).Val(), int64(10))

	ctxCancel, cancel := context.WithCancel(context.Background())
	trimmer.Start(ctxCancel)

	time.Sleep(100 * time.Millisecond)

	cancel()
	trimmer.Wait()
}

func TestRedisStreamTrimmer_PELPendingNotInflated(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

	ctx := context.Background()
	stream := "ad:events:ch"
	group := "processor-ch-group"

	require.NoError(t, redisClient.XGroupCreateMkStream(ctx, stream, group, "0").Err())

	for i := range 30 {
		_, err := redisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]interface{}{"k": i},
		}).Result()
		require.NoError(t, err)
	}

	consumer := "trimmer-smoke"
	read, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    8,
	}).Result()
	require.NoError(t, err)
	require.Len(t, read, 1)
	require.Len(t, read[0].Messages, 8)

	pendingBefore, err := redisClient.XPending(ctx, stream, group).Result()
	require.NoError(t, err)
	require.Greater(t, pendingBefore.Count, int64(0), "consumer must have pending entries before trim")

	trimmer := NewRedisStreamTrimmer(RedisStreamTrimmerConfig{
		RedisShards: []redis.UniversalClient{redisClient},
		Streams:     []string{stream},
		MaxLen:      10,
	})
	trimmer.TrimOnce(ctx)

	assert.LessOrEqual(t, redisClient.XLen(ctx, stream).Val(), int64(10), "trimmer must cap stream length")

	for _, msg := range read[0].Messages {
		require.NoError(t, redisClient.XAck(ctx, stream, group, msg.ID).Err(),
			"in-flight PEL entries must remain ackable after XTRIM")
	}

	pendingAfterAck, err := redisClient.XPending(ctx, stream, group).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pendingAfterAck.Count)
}
