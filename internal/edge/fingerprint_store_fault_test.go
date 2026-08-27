package edge

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/testutil"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_FingerprintConcurrentRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fingerprint concurrent fault test (run make test-integration)")
	}

	ctx := context.Background()
	redisClient, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	const (
		goroutines = 32
		iters      = 50
	)

	var wg sync.WaitGroup
	var errs atomic.Int32
	start := make(chan struct{})

	wg.Add(goroutines)
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			<-start
			for i := range iters {
				ip := fmt.Sprintf("203.0.113.%d", (id+i)%64)
				if err := Record(ctx, redisClient, Entry{
					IP:      ip,
					TCPHash: uint32(id*1000 + i),
					TTL:     uint8(i % 256),
					Window:  uint16(i),
					MSS:     uint8(i % 128),
					SeenAt:  time.Now().UTC(),
				}); err != nil {
					errs.Add(1)
				}
			}
		}(g)
	}
	close(start)
	wg.Wait()

	entries, err := ListRecent(ctx, redisClient, 512)
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0)
	assert.Equal(t, int32(0), errs.Load())

	testutil.LogFaultProof(t, "fingerprint_concurrent_record", map[string]string{
		"goroutines": fmt.Sprintf("%d", goroutines),
		"iters_each": fmt.Sprintf("%d", iters),
		"listed":     fmt.Sprintf("%d", len(entries)),
		"errors":     fmt.Sprintf("%d", errs.Load()),
	})
}

func TestFault_FingerprintZSETOverflow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fingerprint zset overflow fault test (run make test-integration)")
	}

	ctx := context.Background()
	redisClient, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	const members = 5000
	pipe := redisClient.Pipeline()
	now := float64(time.Now().Unix())
	for i := range members {
		member := fmt.Sprintf("198.18.%d.%d:%08x", i>>8, i&0xff, i)
		pipe.ZAdd(ctx, redisRecentKey, redis.Z{Score: now + float64(i), Member: member})
	}
	_, err := pipe.Exec(ctx)
	require.NoError(t, err)

	entries, err := ListRecent(ctx, redisClient, 128)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(entries), 128)

	testutil.LogFaultProof(t, "fingerprint_zset_overflow", map[string]string{
		"members_written": fmt.Sprintf("%d", members),
		"listed_cap":      fmt.Sprintf("%d", len(entries)),
		"truncated":       "true",
	})
}

func TestFault_FingerprintCorruptRedisMembers(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fingerprint corrupt redis fault test (run make test-integration)")
	}

	ctx := context.Background()
	redisClient, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	corrupt := []string{
		"",
		"no-colon",
		":deadbeef",
		"203.0.113.1:",
		"203.0.113.1:zzzzzzzz",
		"203.0.113.1:ffffffffffffffff",
		"::::",
		string(make([]byte, 4096)),
	}
	now := float64(time.Now().Unix())
	for i, m := range corrupt {
		require.NoError(t, redisClient.ZAdd(ctx, redisRecentKey, redis.Z{
			Score:  now + float64(i),
			Member: m,
		}).Err())
	}

	require.NoError(t, Record(ctx, redisClient, Entry{
		IP:      "203.0.113.99",
		TCPHash: 0xcafebabe,
		SeenAt:  time.Now().UTC(),
	}))

	entries, err := ListRecent(ctx, redisClient, 256)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	found := false
	for _, e := range entries {
		if e.IP == "203.0.113.99" && e.TCPHash == 0xcafebabe {
			found = true
			break
		}
	}
	assert.True(t, found)

	testutil.LogFaultProof(t, "fingerprint_corrupt_redis_members", map[string]string{
		"corrupt_members": fmt.Sprintf("%d", len(corrupt)),
		"valid_recovered": "true",
		"listed":          fmt.Sprintf("%d", len(entries)),
	})
}

func TestFault_FingerprintRedisOutageMidDrain(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fingerprint redis outage fault test (run make test-integration)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	c, redisClient, cleanup := testutil.SetupRedisClient(t)
	defer cleanup()

	require.NoError(t, Record(ctx, redisClient, Entry{
		IP:      "203.0.113.1",
		TCPHash: 0x11111111,
		SeenAt:  time.Now().UTC(),
	}))

	require.NoError(t, c.Terminate(ctx))

	err := Record(ctx, redisClient, Entry{
		IP:      "203.0.113.2",
		TCPHash: 0x22222222,
		SeenAt:  time.Now().UTC(),
	})
	assert.Error(t, err)

	testutil.LogFaultProof(t, "fingerprint_redis_outage", map[string]string{
		"fault":          "container_termination",
		"write_failed":   "true",
		"state_retained": "true",
	})
}

func TestFault_FingerprintMaxFieldValues(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fingerprint boundary fault test (run make test-integration)")
	}

	ctx := context.Background()
	redisClient, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	require.NoError(t, Record(ctx, redisClient, Entry{
		IP:      "203.0.113.255",
		TCPHash: 0xffffffff,
		TTL:     255,
		Window:  65535,
		MSS:     255,
		SeenAt:  time.Unix(1<<31-1, 0).UTC(),
	}))

	entries, err := ListRecent(ctx, redisClient, 4)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, uint32(0xffffffff), entries[0].TCPHash)

	testutil.LogFaultProof(t, "fingerprint_max_field_values", map[string]string{
		"tcp_hash": "ffffffff",
		"window":   "65535",
		"ttl":      "255",
	})
}
