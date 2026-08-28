package broker

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestOpenCoordRedis_unixURLAppliesEnvPassword_holdout(t *testing.T) {
	t.Setenv("BROKER_REDIS_PASSWORD", "")
	t.Setenv("REDIS_PASSWORD", "coord-redis-secret")

	redisClient, err := openCoordRedis("unix:///run/ad-event-processor/redis/redis-0.sock?db=0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = redisClient.Close() })

	opts := redisClient.(*redis.Client).Options()
	require.Equal(t, "coord-redis-secret", opts.Password)
	require.Equal(t, "unix", opts.Network)
	require.Equal(t, "/run/ad-event-processor/redis/redis-0.sock", opts.Addr)
}

func TestOpenCoordRedis_tcpURLAppliesEnvPassword_holdout(t *testing.T) {
	t.Setenv("BROKER_REDIS_PASSWORD", "")
	t.Setenv("REDIS_PASSWORD", "coord-redis-secret")

	redisClient, err := openCoordRedis("redis://127.0.0.1:6379/0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = redisClient.Close() })

	opts := redisClient.(*redis.Client).Options()
	require.Equal(t, "coord-redis-secret", opts.Password)
}
