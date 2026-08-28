package controlplane

import (
	"context"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/identity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

const leaseFaultContainerStopTimeout = 10 * time.Second

func stopLeasePGContainer(t *testing.T, infra *database.TestDBInfra) {
	t.Helper()
	timeout := leaseFaultContainerStopTimeout
	require.NoError(t, infra.PGContainer.Stop(context.Background(), &timeout))
}

func startLeasePGContainer(t *testing.T, infra *database.TestDBInfra) {
	t.Helper()
	require.NoError(t, infra.PGContainer.Start(context.Background()))
}

func refreshLeasePGPool(t *testing.T, infra *database.TestDBInfra) {
	t.Helper()
	ctx := context.Background()
	infra.Pool.Close()
	connStr, err := infra.PGContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	infra.Pool = pool
	require.Eventually(t, func() bool {
		return pool.Ping(ctx) == nil
	}, 30*time.Second, 200*time.Millisecond)
}

func integrationTestAuth(t *testing.T, redisClient redis.UniversalClient, cfg *config.Config) (*AuthMiddleware, identity.Maker) {
	t.Helper()
	if cfg.TokenSymmetricKey == "" {
		cfg.TokenSymmetricKey = "01234567890123456789012345678901"
	}
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		t.Fatalf("token maker: %v", err)
	}
	return NewAuthMiddleware(tokenMaker, redisClient, cfg, nil), tokenMaker
}

func withSessionUser(req *http.Request, tokenMaker identity.Maker, role string, customerID uuid.UUID) {
	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), role, customerID, time.Hour)
	if err != nil {
		panic(err)
	}
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
}

func withAdminAPIKey(req *http.Request, cfg *config.Config) {
	req.Header.Set("X-Admin-API-Key", string(cfg.AdminAPIKey))
}

func newBareService(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) *Service {
	return NewBareServiceForTest(t, pool, redisShards, cfg)
}

func isDeadlock(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "deadlock detected") || strings.Contains(msg, "40P01")
}

type slowRedisClient struct {
	redis.UniversalClient
	delay time.Duration
}

func (c *slowRedisClient) Pipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Pipelined(ctx, fn)
}

func (c *slowRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Set(ctx, key, value, expiration)
}

func (c *slowRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.SAdd(ctx, key, members...)
}

func (c *slowRedisClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.SRem(ctx, key, members...)
}

func (c *slowRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Del(ctx, keys...)
}

func (c *slowRedisClient) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Publish(ctx, channel, message)
}

func (c *slowRedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.HSet(ctx, key, values...)
}

func (c *slowRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.UniversalClient.Get(ctx, key)
}
