package testutil

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func SetupRedis(t testing.TB) (rdb redis.UniversalClient, cleanup func()) {
	t.Helper()
	_, rdb, cleanup = SetupRedisContainer(t)
	return
}

func SetupRedisContainer(t testing.TB) (container testcontainers.Container, rdb redis.UniversalClient, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %s", err)
	}

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get redis endpoint: %s", err)
	}

	rdb = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{endpoint},
	})
	container = redisContainer

	cleanup = func() {
		_ = rdb.Close()
		_ = redisContainer.Terminate(ctx)
	}
	return
}
