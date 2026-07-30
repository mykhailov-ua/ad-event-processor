package testutil

import (
	"strings"
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

func NewInjectionEnv(t testing.TB) (*InjectionEnv, func()) {
	t.Helper()

	pgContainer, pool, cleanupPG := SetupAdsPostgresContainer(t)
	redisContainer, rdb, cleanupRedis := SetupRedisContainer(t)

	env := &InjectionEnv{
		Pool:           pool,
		Redis:          rdb,
		PGContainer:    pgContainer,
		RedisContainer: redisContainer,
	}
	cleanup := func() {
		cleanupRedis()
		cleanupPG()
	}
	return env, cleanup
}

func LogFaultProof(t testing.TB, scenario string, attrs map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("fault_proof fault=")
	b.WriteString(scenario)
	for k, v := range attrs {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
	}
	t.Log(b.String())
}
