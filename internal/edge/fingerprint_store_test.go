package edge

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordAndListRecent(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: redis testcontainers (run make test-integration)")
	}
	redisClient, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	seenAt := time.Unix(1_700_000_000, 0).UTC()

	require.NoError(t, Record(ctx, redisClient, Entry{
		IP:          "203.0.113.10",
		TCPHash:     0xdeadbeef,
		TTL:         64,
		Window:      64240,
		MSS:         44,
		TCPOptTrace: "nop,nop,sackok,mss:1460",
		SeenAt:      seenAt,
	}))

	got, err := redisClient.HGet(ctx, "edge:tcp_fp:ip:203.0.113.10", "tcp_opt_trace").Result()
	require.NoError(t, err)
	assert.Equal(t, "nop,nop,sackok,mss:1460", got)

	entries, err := ListRecent(ctx, redisClient, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "203.0.113.10", entries[0].IP)
	assert.Equal(t, uint32(0xdeadbeef), entries[0].TCPHash)
}
