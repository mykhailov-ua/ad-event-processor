package controlplane

import (
	"espx/pkg/faultproof"

	"context"
	"io"
	"net"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_ShardOrchestrator_NoFalseMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb0, cleanup0 := database.SetupTestRedis(t)
	defer cleanup0()
	rdb1, cleanup1 := database.SetupTestRedis(t)
	defer cleanup1()

	cfg := &config.Config{SlotMigrationEnabled: false}
	svc := NewService(pool, []redis.UniversalClient{rdb0, rdb1}, domain.NewStaticSlotSharder(2), cfg)
	defer svc.Close()

	var campID uuid.UUID
	for {
		campID = uuid.New()
		if domain.CampaignSlotIndex(campID)%2 == 0 {
			break
		}
	}
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "SO Cust 1", 1_000_000, "USD"))

	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'so-test-1', 1000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campID), domain.ToUUID(customerID))
	require.NoError(t, err)

	provider := &mockShardMetricsProvider{
		metrics: map[int16]ShardMetrics{
			0: {ShardID: 0, CPUUsage: 40.0, MemoryPct: 30.0, OpsPerSec: 10000},
			1: {ShardID: 1, CPUUsage: 10.0, MemoryPct: 15.0, OpsPerSec: 1000},
		},
	}

	orchestrator := NewShardOrchestrator(svc, provider, 100*time.Millisecond)
	orchestrator.scaleThreshold = 0.85
	orchestrator.overloadLimit = 10 * time.Millisecond

	for i := 0; i < 5; i++ {
		orchestrator.tick(ctx)
		time.Sleep(10 * time.Millisecond)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM campaign_routing").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	faultproof.Log(t, "orchestrator_no_false_migrate", map[string]string{
		"subsystem":     "shard_orchestrator",
		"max_ema":       "0.40",
		"threshold":     "0.85",
		"false_migrate": "false",
	})
}

func TestFault_ShardOrchestrator_CampaignRoutingMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb0, cleanup0 := database.SetupTestRedis(t)
	defer cleanup0()
	rdb1, cleanup1 := database.SetupTestRedis(t)
	defer cleanup1()

	cfg := &config.Config{SlotMigrationEnabled: false}
	svc := NewService(pool, []redis.UniversalClient{rdb0, rdb1}, domain.NewStaticSlotSharder(2), cfg)
	defer svc.Close()

	var campID uuid.UUID
	for {
		campID = uuid.New()
		if domain.CampaignSlotIndex(campID)%2 == 0 {
			break
		}
	}
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "SO Cust 2", 1_000_000, "USD"))

	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'so-test-2', 1000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campID), domain.ToUUID(customerID))
	require.NoError(t, err)

	key := "budget:campaign:" + campID.String()
	require.NoError(t, rdb0.Set(ctx, key, "850000", 0).Err())

	provider := &mockShardMetricsProvider{
		metrics: map[int16]ShardMetrics{
			0: {ShardID: 0, CPUUsage: 95.0, MemoryPct: 90.0, OpsPerSec: 60000},
			1: {ShardID: 1, CPUUsage: 10.0, MemoryPct: 15.0, OpsPerSec: 1000},
		},
	}

	orchestrator := NewShardOrchestrator(svc, provider, 10*time.Millisecond)
	orchestrator.scaleThreshold = 0.85
	orchestrator.overloadLimit = 20 * time.Millisecond
	orchestrator.cooldown = 0

	orchestrator.tick(ctx)
	time.Sleep(30 * time.Millisecond)
	orchestrator.tick(ctx)

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM campaign_routing WHERE campaign_id = $1", domain.ToUUID(campID)).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	exists, err := rdb1.Exists(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), exists)

	existsSource, err := rdb0.Exists(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), existsSource)

	faultproof.Log(t, "campaign_routing_migration", map[string]string{
		"subsystem":         "shard_orchestrator",
		"source_shard":      "0",
		"target_shard":      "1",
		"migration_success": "true",
		"keys_drained":      "true",
	})
}

func TestFault_ShardOrchestrator_RoutingEpochRace(t *testing.T) {
	t.Parallel()
	secret := []byte("epoch-race-secret")
	sh := domain.NewStaticSlotSharder(4)
	sh.SwapSnapshot(1, nil, 10)

	var oldHdr domain.TCPControlHeader
	oldHdr.MsgType = domain.TCPMsgSnapshot
	oldHdr.RoutingEpoch = 10
	oldHdr.SlotMapVersion = 1
	var oldFrame [64]byte
	_, err := domain.EncodeTCPControlFrame(oldFrame[:], secret, &oldHdr, nil)
	require.NoError(t, err)

	var newHdr domain.TCPControlHeader
	newHdr.MsgType = domain.TCPMsgSnapshot
	newHdr.RoutingEpoch = 11
	newHdr.SlotMapVersion = 2
	var newFrame [64]byte
	_, err = domain.EncodeTCPControlFrame(newFrame[:], secret, &newHdr, nil)
	require.NoError(t, err)

	apply := func(frame []byte) int64 {
		var hdr domain.TCPControlHeader
		_, err := domain.DecodeTCPControlFrame(frame, secret, &hdr)
		require.NoError(t, err)
		if hdr.RoutingEpoch > sh.Snapshot().MigrationGen {
			prev := sh.Snapshot()
			sh.SwapSnapshot(hdr.SlotMapVersion, &prev.Table, hdr.RoutingEpoch)
		}
		return sh.Snapshot().MigrationGen
	}

	require.Equal(t, int64(11), apply(newFrame[:]))
	require.Equal(t, int64(11), apply(oldFrame[:]))

	faultproof.Log(t, "routing_epoch_race", map[string]string{
		"applied_epoch": "11",
		"stale_blocked": "true",
	})
}

func requestTCPSnapshotACK(t *testing.T, ctx context.Context, addr string, secret []byte, trackerID uint32, sh *domain.StaticSlotSharder) {
	t.Helper()
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	var reqHdr domain.TCPControlHeader
	reqHdr.MsgType = domain.TCPMsgSnapshotRequest
	reqHdr.TrackerID = trackerID
	if sh != nil {
		reqHdr.SlotMapVersion = sh.ActiveVersion()
		reqHdr.RoutingEpoch = sh.Snapshot().MigrationGen
	}
	var reqBuf [domain.TCPControlHeaderSize]byte
	_, err = domain.EncodeTCPControlFrame(reqBuf[:], secret, &reqHdr, nil)
	require.NoError(t, err)
	_, err = conn.Write(reqBuf[:])
	require.NoError(t, err)

	var frame [4096]byte
	n, err := io.ReadAtLeast(conn, frame[:], domain.TCPControlHeaderSize)
	require.NoError(t, err)
	for n < cap(frame) {
		m, rerr := conn.Read(frame[n:])
		n += m
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			require.NoError(t, rerr)
		}
		if m == 0 {
			break
		}
	}
	var hdr domain.TCPControlHeader
	_, err = domain.DecodeTCPControlFrame(frame[:n], secret, &hdr)
	require.NoError(t, err)
	require.Equal(t, domain.TCPMsgSnapshot, hdr.MsgType)

	ack := domain.TCPAckPayload{
		TrackerID:      trackerID,
		AppliedEpoch:   hdr.RoutingEpoch,
		AppliedSlotVer: hdr.SlotMapVersion,
	}
	var body [16]byte
	require.NotZero(t, domain.EncodeTCPAckPayload(body[:], &ack))
	var ackHdr domain.TCPControlHeader
	ackHdr.MsgType = domain.TCPMsgAck
	ackHdr.TrackerID = trackerID
	ackHdr.RoutingEpoch = hdr.RoutingEpoch
	ackHdr.SlotMapVersion = hdr.SlotMapVersion
	var ackFrame [80]byte
	ackN, err := domain.EncodeTCPControlFrame(ackFrame[:], secret, &ackHdr, body[:])
	require.NoError(t, err)
	_, err = conn.Write(ackFrame[:ackN])
	require.NoError(t, err)
}

func TestFault_TCP_SnapshotHMACACK(t *testing.T) {
	secret := []byte("tcp-hmac-secret")
	cfg := &config.Config{
		TCPControlEnabled:    true,
		TCPControlHMACSecret: config.Secret(secret),
	}
	sh := domain.NewStaticSlotSharder(2)
	srv := NewTCPControlServer(cfg, nil, sh, 2)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	srv.ln = ln

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.acceptLoop(ctx)

	requestTCPSnapshotACK(t, ctx, ln.Addr().String(), secret, 1, sh)

	faultproof.Log(t, "tcp_snapshot_hmac_ack", map[string]string{
		"subsystem": "tcp_control",
		"ack":       "true",
	})
}
