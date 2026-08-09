package e2e_test

import (
	"context"
	"testing"
	"time"

	"espx/internal/database"
	db "espx/internal/domain/db"
	"espx/internal/ingestion"
	"espx/internal/ingestion/pb"
	"espx/internal/testutil"
	"espx/pkg/broker/client"
	bserver "espx/pkg/broker/server"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type settlementSnapshot struct {
	Clicks      int64
	Impressions int64
}

// TestE2E_BrokerPGSettlementParity verifies Redis _pg and broker _pg_broker consumers
// apply identical campaign_stats settlement for the same AdStreamEvent mix.
func TestE2E_BrokerPGSettlementParity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping broker pg settlement parity e2e")
	}

	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()

	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queries := db.New(pool)
	partManager := database.NewPartitionManager(pool, 7, 2)
	require.NoError(t, partManager.Run(ctx))

	customerID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)", customerID, "Parity Customer", 1_000_000_000)
	require.NoError(t, err)

	campaignRedis := uuid.New()
	campaignBroker := uuid.New()
	for _, campID := range []uuid.UUID{campaignRedis, campaignBroker} {
		_, err = pool.Exec(ctx,
			"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
			campID, "Parity Campaign", "ACTIVE", customerID, 100_000_000,
		)
		require.NoError(t, err)
	}

	eventMix := []struct {
		suffix    string
		eventType string
	}{
		{"click-1", "click"},
		{"click-2", "click"},
		{"imp-1", "impression"},
		{"imp-2", "impression"},
	}

	const streamName = "ad:events:stream"
	const baseGroup = "ad:processor:group"

	pgStore := ingestion.NewPostgresStore(queries, time.Second)
	settleStore := ingestion.NewSettlementStore(pgStore, true)

	settleW := ingestion.NewSettlementWorker(
		settleStore, rdb, streamName, baseGroup+"_pg", "e2e-redis",
		1, len(eventMix),
		50*time.Millisecond, time.Second,
		50*time.Millisecond, time.Second,
		3,
		5*time.Second, 5*time.Second,
	)
	settleW.Start(ctx)
	defer func() {
		settleW.Close()
		_ = settleW.Wait(context.Background())
	}()

	for _, e := range eventMix {
		clickID := "redis-" + e.suffix
		require.NoError(t, xaddAdStreamEvent(ctx, rdb, streamName, campaignRedis, clickID, e.eventType))
	}

	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 8*1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	brokerCfg := ingestion.BrokerConsumerConfig{
		BrokerAddr: srv.Addr(),
		Topic:      "tracker-logs",
		Group:      baseGroup + "_pg_broker",
		BatchSize:  len(eventMix),
		FlushInt:   50 * time.Millisecond,
		MaxBytes:   1024 * 1024,
		Timeout:    2 * time.Second,
		IdleWait:   20 * time.Millisecond,
		ShadowMode: false,
	}
	brokerConsumer := ingestion.NewBrokerStreamConsumer(pgStore, brokerCfg, time.Second, 50*time.Millisecond, time.Second, 3)
	brokerConsumer.Start(ctx)
	defer brokerConsumer.Close()

	producer := client.NewClient(srv.Addr(), 2*time.Second)
	require.NoError(t, producer.Connect())
	for _, e := range eventMix {
		clickID := "broker-" + e.suffix
		require.NoError(t, produceAdStreamEvent(producer, "tracker-logs", campaignBroker, clickID, e.eventType))
	}
	require.NoError(t, producer.Close())

	var redisSnap, brokerSnap settlementSnapshot
	require.Eventually(t, func() bool {
		var err error
		redisSnap, err = readSettlementSnapshot(ctx, pool, campaignRedis)
		if err != nil || redisSnap.Clicks != 2 || redisSnap.Impressions != 2 {
			return false
		}
		brokerSnap, err = readSettlementSnapshot(ctx, pool, campaignBroker)
		return err == nil && brokerSnap.Clicks == 2 && brokerSnap.Impressions == 2
	}, 8*time.Second, 100*time.Millisecond)

	assert.Equal(t, redisSnap, brokerSnap, "Redis _pg and broker _pg_broker must produce identical settlement stats")

	var brokerEventRows int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM events WHERE campaign_id = $1", campaignBroker,
	).Scan(&brokerEventRows))
	assert.Equal(t, len(eventMix), brokerEventRows, "broker _pg_broker should persist operational events rows")

	var redisEventRows int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM events WHERE campaign_id = $1", campaignRedis,
	).Scan(&redisEventRows))
	assert.Equal(t, 0, redisEventRows, "Redis _pg stats-only path must not insert events rows")
}

func readSettlementSnapshot(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) (settlementSnapshot, error) {
	var snap settlementSnapshot
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(clicks_count), 0), COALESCE(SUM(impressions_count), 0)
		 FROM campaign_stats WHERE campaign_id = $1`,
		campaignID,
	).Scan(&snap.Clicks, &snap.Impressions)
	return snap, err
}

func marshalAdStreamEvent(campaignID uuid.UUID, clickID, eventType string) ([]byte, error) {
	rec := &pb.AdStreamEvent{
		CreatedAtUnix: time.Now().Unix(),
		CampaignId:    campaignID[:],
		ClickId:       []byte(clickID),
		EventType:     []byte(eventType),
		Ip:            []byte("203.0.113.42"),
		Ua:            []byte("parity-agent"),
		UserId:        []byte("parity-user"),
	}
	data := make([]byte, rec.SizeVT())
	n, err := rec.MarshalToSizedBufferVT(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func xaddAdStreamEvent(ctx context.Context, rdb redis.UniversalClient, stream string, campaignID uuid.UUID, clickID, eventType string) error {
	data, err := marshalAdStreamEvent(campaignID, clickID, eventType)
	if err != nil {
		return err
	}
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{"d": string(data)},
	}).Err()
}

func produceAdStreamEvent(producer *client.Client, topic string, campaignID uuid.UUID, clickID, eventType string) error {
	data, err := marshalAdStreamEvent(campaignID, clickID, eventType)
	if err != nil {
		return err
	}
	_, err = producer.Produce(topic, 0, data)
	return err
}
