package testutil

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type InjectionEnv struct {
	Pool           *pgxpool.Pool
	Redis          redis.UniversalClient
	PGContainer    *postgres.PostgresContainer
	RedisContainer testcontainers.Container
}

func NewInjectionEnv(t testing.TB) (env *InjectionEnv, cleanup func()) {
	t.Helper()

	pgContainer, pool, cleanupPG := SetupAdsPostgresContainer(t)
	redisContainer, rdb, cleanupRedis := SetupRedisContainer(t)

	env = &InjectionEnv{
		Pool:           pool,
		Redis:          rdb,
		PGContainer:    pgContainer,
		RedisContainer: redisContainer,
	}
	cleanup = func() {
		cleanupRedis()
		cleanupPG()
	}
	return
}
