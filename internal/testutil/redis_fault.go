package testutil

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const redisShardStopTimeout = 10 * time.Second

type RedisShardFaultInfra struct {
	Clients    []*redis.Client
	Containers []testcontainers.Container
	Breakers   []*database.RedisBreaker
	cleanups   []func()
}

func SetupRedisShards(t testing.TB, n int) []redis.UniversalClient {
	t.Helper()
	shards := make([]redis.UniversalClient, n)
	for i := range shards {
		rdb, cleanup := SetupRedis(t)
		t.Cleanup(cleanup)
		shards[i] = rdb
	}
	return shards
}

func SetupRedisShardsFault(t testing.TB, n int) *RedisShardFaultInfra {
	t.Helper()
	infra := &RedisShardFaultInfra{
		Clients:    make([]*redis.Client, n),
		Containers: make([]testcontainers.Container, n),
		Breakers:   make([]*database.RedisBreaker, n),
	}
	for i := range infra.Clients {
		c, rdb, cleanup := SetupRedisClient(t)
		infra.Containers[i] = c
		infra.Clients[i] = rdb
		infra.cleanups = append(infra.cleanups, cleanup)

		breaker := database.NewRedisBreaker(3, 2, 300*time.Millisecond)
		infra.Breakers[i] = breaker
		infra.Clients[i].AddHook(database.NewRedisCircuitBreakerHook(breaker))

		require.NoError(t, infra.Clients[i].Ping(context.Background()).Err())
	}
	t.Cleanup(infra.cleanup)
	return infra
}

func (infra *RedisShardFaultInfra) UniversalClients() []redis.UniversalClient {
	out := make([]redis.UniversalClient, len(infra.Clients))
	for i, c := range infra.Clients {
		out[i] = c
	}
	return out
}

func (infra *RedisShardFaultInfra) cleanup() {
	for _, fn := range infra.cleanups {
		fn()
	}
}

func (infra *RedisShardFaultInfra) ReplaceShardClient(t testing.TB, idx int, rdbs []redis.UniversalClient) {
	t.Helper()
	ctx := context.Background()
	endpoint, err := infra.Containers[idx].Endpoint(ctx, "")
	require.NoError(t, err)
	_ = infra.Clients[idx].Close()

	client := redis.NewClient(&redis.Options{
		Addr:         endpoint,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
	})
	breaker := database.NewRedisBreaker(3, 2, 300*time.Millisecond)
	client.AddHook(database.NewRedisCircuitBreakerHook(breaker))

	infra.Clients[idx] = client
	infra.Breakers[idx] = breaker
	if rdbs != nil && idx < len(rdbs) {
		rdbs[idx] = client
	}
	require.NoError(t, client.Ping(ctx).Err())
}

func SetupRedisClient(t testing.TB) (testcontainers.Container, *redis.Client, func()) {
	t.Helper()
	ctx := context.Background()
	c, rdb, cleanup := SetupRedisContainer(t)

	endpoint, err := c.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get redis endpoint: %s", err)
	}
	_ = rdb.Close()

	client := redis.NewClient(&redis.Options{
		Addr:         endpoint,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
	})

	return c, client, func() {
		_ = client.Close()
		cleanup()
	}
}

func WaitRedisContainerReady(t testing.TB, c testcontainers.Container) {
	t.Helper()
	require.Eventually(t, func() bool {
		ctx := context.Background()
		endpoint, err := c.Endpoint(ctx, "")
		if err != nil {
			return false
		}
		probe := redis.NewClient(&redis.Options{Addr: endpoint, ReadTimeout: time.Second})
		defer func() { _ = probe.Close() }()
		return probe.Ping(ctx).Err() == nil
	}, 30*time.Second, 200*time.Millisecond)
}

func StopRedisShardContainer(t testing.TB, c testcontainers.Container) {
	t.Helper()
	timeout := redisShardStopTimeout
	require.NoError(t, c.Stop(context.Background(), &timeout))
}

func StartRedisShardContainer(t testing.TB, c testcontainers.Container) {
	t.Helper()
	require.NoError(t, c.Start(context.Background()))
}

func WaitRedisClientReady(t testing.TB, rdb redis.UniversalClient) {
	t.Helper()
	require.Eventually(t, func() bool {
		return rdb.Ping(context.Background()).Err() == nil
	}, 30*time.Second, 100*time.Millisecond)
}

func TripRedisBreaker(t testing.TB, rdb redis.UniversalClient, breaker *database.RedisBreaker) {
	t.Helper()
	ctx := context.Background()
	require.Eventually(t, func() bool {
		for range 3 {
			_ = rdb.Ping(ctx).Err()
		}
		return breaker.State() == database.CircuitOpen
	}, 10*time.Second, 50*time.Millisecond)
}

func WaitRedisBreakerClosed(t testing.TB, rdb redis.UniversalClient, breaker *database.RedisBreaker) {
	t.Helper()
	ctx := context.Background()
	require.Eventually(t, func() bool {
		_ = rdb.Ping(ctx).Err()
		return breaker.State() == database.CircuitClosed
	}, 15*time.Second, 100*time.Millisecond)
}

func LatencyBudget(baseline time.Duration) time.Duration {
	if baseline < time.Millisecond {
		baseline = time.Millisecond
	}
	return baseline * 2
}
