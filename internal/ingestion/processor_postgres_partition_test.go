package ingestion

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moby/moby/api/types/container"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const processorPGProxyPort = "5432/tcp"

type adsProcessorPartitionInfra struct {
	*adsFaultInfra
	ProcessorContainer testcontainers.Container
	pgProxyConnStr     string
}

func TestFault_AdsProcessorPGNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupAdsProcessorPartitionInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra.adsFaultInfra, "ads-fault-processor-pg-partition")
	defer stack.Close(t)

	ctx := context.Background()
	const baselineTracks = 4
	for range baselineTracks {
		require.Equal(t, http.StatusAccepted, postFaultClick(t, stack.Handler, stack.CampaignID))
	}
	waitFaultStreamDrained(t, infra.Redis, stack.Stream, stack.Stream+"-group", stack.CampaignID, infra.Pool, baselineTracks)

	rowsBaseline := countFaultCampaignEvents(t, infra.Pool, stack.CampaignID)
	streamBeforePartition, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)

	blockProcessorPGPartition(t, infra.ProcessorContainer)
	t.Cleanup(func() { unblockProcessorPGPartition(t, infra.ProcessorContainer) })

	requirePGContainerAlive(t, infra.PGContainer)
	require.Eventually(t, func() bool {
		return verifyProcessorPGPartitionActive(ctx, infra.pgProxyConnStr)
	}, 20*time.Second, 500*time.Millisecond, "processor path to postgres must fail after iptables DROP")

	const bufferedTracks = 8
	acceptedDuringPartition := 0
	for range bufferedTracks {
		if postFaultClick(t, stack.Handler, stack.CampaignID) == http.StatusAccepted {
			acceptedDuringPartition++
		}
	}
	require.Equal(t, bufferedTracks, acceptedDuringPartition, "tracker must keep accepting during PG partition")

	require.Eventually(t, func() bool {
		return stack.Consumer.CircuitBreakerState() == CircuitOpen
	}, 25*time.Second, 200*time.Millisecond, "consumer circuit must open when processor cannot reach PG")

	streamDuringPartition, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)
	assert.Greater(t, streamDuringPartition, streamBeforePartition,
		"stream must grow while processor is partitioned from PG")
	backpressureActive := stack.Consumer.CircuitBreakerState() == CircuitOpen &&
		streamDuringPartition > streamBeforePartition

	unblockProcessorPGPartition(t, infra.ProcessorContainer)
	stack.Consumer.Close()
	_ = stack.Consumer.Wait(ctx)
	infra.refreshPGPoolViaProxy(t)
	stack.replaceConsumer(t, infra.adsFaultInfra)

	expectedRows := rowsBaseline + int64(acceptedDuringPartition)
	recovered := false
	require.Eventually(t, func() bool {
		recovered = countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) >= expectedRows
		return recovered
	}, 45*time.Second, 200*time.Millisecond, "consumer must drain backlog after partition lift")

	finalRows := countFaultCampaignEvents(t, infra.Pool, stack.CampaignID)
	distinctClickIDs := countDistinctFaultClickIDs(t, infra.Pool, stack.CampaignID)
	idempotencyOK := finalRows == expectedRows && distinctClickIDs == expectedRows
	require.True(t, idempotencyOK,
		"exactly-once in PG: rows=%d distinct_click_ids=%d expected=%d",
		finalRows, distinctClickIDs, expectedRows)

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, stack.CampaignID)

	faultproof.Log(t, "processor_pg_partition", map[string]string{
		"subsystem":            "ads_processor",
		"baseline_ok":          "true",
		"backpressure_active":  strconv.FormatBool(backpressureActive),
		"idempotency_verified": strconv.FormatBool(idempotencyOK),
		"recovered":            strconv.FormatBool(recovered),
		"rows_expected":        itoaAdsFault(int(expectedRows)),
		"rows_final":           itoaAdsFault(int(finalRows)),
		"stream_delta":         itoaAdsFault(int(streamDuringPartition - streamBeforePartition)),
		"fault_verify":         "processor_iptables_output_drop_5432",
	})
}

func setupAdsProcessorPartitionInfra(t *testing.T) (*adsProcessorPartitionInfra, func()) {
	t.Helper()
	ctx := context.Background()

	nw, err := tcnetwork.New(ctx)
	require.NoError(t, err)

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("ads_fault_db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		tcnetwork.WithNetwork([]string{"postgres"}, nw),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	processorContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "alpine:3.20",
			ExposedPorts: []string{processorPGProxyPort},
			Networks:     []string{nw.Name},
			NetworkAliases: map[string][]string{
				nw.Name: {"processor"},
			},
			Cmd: []string{
				"sh", "-c",
				"apk add --no-cache socat iptables >/dev/null 2>&1 && exec socat TCP-LISTEN:5432,fork,reuseaddr TCP:postgres:5432",
			},
			WaitingFor: wait.ForListeningPort(processorPGProxyPort).WithStartupTimeout(30 * time.Second),
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.CapAdd = append(hc.CapAdd, "NET_ADMIN")
			},
		},
		Started: true,
	})
	require.NoError(t, err)

	proxyHost, err := processorContainer.Host(ctx)
	require.NoError(t, err)
	proxyPort, err := processorContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)
	pgProxyConnStr := fmt.Sprintf(
		"postgres://user:pass@%s:%s/ads_fault_db?sslmode=disable",
		proxyHost, proxyPort.Port(),
	)

	pool, err := pgxpool.New(ctx, pgProxyConnStr)
	require.NoError(t, err)
	applyAdsMigrations(t, pool)

	redisContainer, err := rediscontainer.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{endpoint}})
	require.NoError(t, redisClient.Ping(ctx).Err())

	base := &adsFaultInfra{
		Pool:           pool,
		Redis:          redisClient,
		Queries:        db.New(pool),
		PGContainer:    pgContainer,
		RedisContainer: redisContainer,
	}

	infra := &adsProcessorPartitionInfra{
		adsFaultInfra:      base,
		ProcessorContainer: processorContainer,
		pgProxyConnStr:     pgProxyConnStr,
	}

	cleanup := func() {
		_ = redisClient.Close()
		pool.Close()
		_ = processorContainer.Terminate(ctx)
		_ = redisContainer.Terminate(ctx)
		_ = pgContainer.Terminate(ctx)
		_ = nw.Remove(ctx)
	}
	return infra, cleanup
}

func (infra *adsProcessorPartitionInfra) refreshPGPoolViaProxy(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	infra.Pool.Close()
	pool, err := pgxpool.New(ctx, infra.pgProxyConnStr)
	require.NoError(t, err)
	infra.Pool = pool
	infra.Queries = db.New(pool)
	waitAdsPGReady(t, infra.Pool)
}

func verifyProcessorPGPartitionActive(ctx context.Context, connStr string) bool {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return false
	}
	cfg.ConnConfig.ConnectTimeout = 2 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return true
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return pool.Ping(pingCtx) != nil
}

func blockProcessorPGPartition(t *testing.T, processor testcontainers.Container) {
	t.Helper()
	ctx := context.Background()
	exitCode, _, err := processor.Exec(ctx, []string{
		"sh", "-c", "iptables -A OUTPUT -p tcp --dport 5432 -j DROP",
	})
	require.NoError(t, err)
	require.Zero(t, exitCode, "iptables DROP must succeed in processor container")
}

func unblockProcessorPGPartition(t *testing.T, processor testcontainers.Container) {
	t.Helper()
	ctx := context.Background()
	_, _, _ = processor.Exec(ctx, []string{
		"sh", "-c", "iptables -D OUTPUT -p tcp --dport 5432 -j DROP",
	})
}

func requirePGContainerAlive(t *testing.T, pgContainer *postgres.PostgresContainer) {
	t.Helper()
	ctx := context.Background()
	exitCode, _, err := pgContainer.Exec(ctx, []string{
		"pg_isready", "-U", "user", "-d", "ads_fault_db",
	})
	require.NoError(t, err)
	require.Zero(t, exitCode, "postgres must stay up during network partition (not terminated)")
}

func countDistinctFaultClickIDs(t *testing.T, pool *pgxpool.Pool, campaignID uuid.UUID) int64 {
	t.Helper()
	var n int64
	err := pool.QueryRow(context.Background(),
		`SELECT count(DISTINCT click_id) FROM events WHERE campaign_id = $1`, campaignID,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

func (s *adsIngestStack) replaceConsumer(t *testing.T, infra *adsFaultInfra) {
	t.Helper()
	store := NewPostgresStore(infra.Queries, 1*time.Second)
	s.Consumer = NewStreamConsumer(store, infra.Redis, s.Stream, s.Stream+"-group", s.Stream+"-c1",
		s.cfg.EventBatchSize, s.cfg.MaxWorkers,
		100*time.Millisecond, 1*time.Second,
		100*time.Millisecond, 5*time.Second,
		3, 5*time.Minute, 1*time.Second)
	s.Consumer.Start(s.ctx)
}

func waitFaultStreamDrained(t *testing.T, redisClient redis.UniversalClient, stream, group string, campaignID uuid.UUID, pool *pgxpool.Pool, wantEvents int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		if countFaultCampaignEvents(t, pool, campaignID) < wantEvents {
			return false
		}
		pending, err := redisClient.XPending(context.Background(), stream, group).Result()
		return err == nil && pending.Count == 0
	}, 20*time.Second, 100*time.Millisecond, "consumer must drain stream before partition")
}
