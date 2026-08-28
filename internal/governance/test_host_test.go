package governance

import (
	"context"
	"sync"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type testHost struct {
	pool        *pgxpool.Pool
	redisShards []redis.UniversalClient
	sharder     domain.Sharder
	cfg         *config.Config
}

func newTestHost(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) *testHost {
	t.Helper()
	if cfg == nil {
		cfg = &config.Config{}
	}
	shardCount := len(redisShards)
	if shardCount == 0 {
		shardCount = 1
	}
	return &testHost{
		pool:        pool,
		redisShards: redisShards,
		sharder:     domain.NewStaticSlotSharder(shardCount),
		cfg:         cfg,
	}
}

func (h *testHost) Pool() *pgxpool.Pool { return h.pool }

func (h *testHost) RedisShards() []redis.UniversalClient { return h.redisShards }

func (h *testHost) Sharder() domain.Sharder { return h.sharder }

func (h *testHost) Config() *config.Config { return h.cfg }

func (h *testHost) AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
}

type mockRedisForQuota struct {
	redis.UniversalClient
	mu   sync.Mutex
	data map[string]int64
}

func newMockRedisForQuota() *mockRedisForQuota {
	return &mockRedisForQuota{data: make(map[string]int64)}
}

func (m *mockRedisForQuota) getVal(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[key]
}

func payloadFromOutbox(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id int64) []byte {
	t.Helper()
	var payload []byte
	require.NoError(t, pool.QueryRow(ctx, `SELECT payload FROM outbox_events WHERE id = $1`, id).Scan(&payload))
	return payload
}

type stubQuorum struct{}

func (q stubQuorum) ObserveShard(context.Context, int, redis.UniversalClient) {}

func (q stubQuorum) DeadShardConfirmed(int) bool { return false }
