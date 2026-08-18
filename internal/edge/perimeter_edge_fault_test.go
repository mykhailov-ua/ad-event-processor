package edge

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const edgeBlacklistSyncInterval = 5 * time.Second

func TestFault_EdgePhase1BlocksBlacklistedIP(t *testing.T) {
	if testing.Short() {
		t.Skip("edge fault integration test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	const blockedIP = "203.0.113.50"
	require.NoError(t, rdb.SAdd(ctx, redisKeyBlacklistManual, blockedIP).Err())

	cache := NewBlacklistCache(defaultStaleSec)
	require.NoError(t, cache.SyncFromRedis(ctx, rdb))

	var metrics Metrics
	now := time.Now().Unix()

	outcome := cache.Phase1Check(blockedIP, now, &metrics)
	assert.Equal(t, Phase1Blocked403, outcome)
	assert.Equal(t, int64(1), metrics.BlockedIP)
	assert.Equal(t, int64(0), metrics.BodyRead, "phase-1 must not read body")
	assert.Equal(t, int64(0), metrics.Phase1Pass)

	legitOutcome := cache.Phase1Check("198.51.100.1", now, &metrics)
	assert.Equal(t, Phase1Pass, legitOutcome)
	assert.Equal(t, int64(1), metrics.Phase1Pass)

	faultproof.Log(t, "edge_phase1_blacklist", map[string]string{
		"blocked_before_body": "true",
		"body_read_total":     strconv.FormatInt(metrics.BodyRead, 10),
		"blocked_ip_total":    strconv.FormatInt(metrics.BlockedIP, 10),
		"harness":             "go_perimeter_mirror",
	})
}

func TestFault_EdgeBlacklistPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("edge fault integration test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	const newIP = "198.51.100.77"
	cache := NewBlacklistCache(defaultStaleSec)
	require.NoError(t, cache.SyncFromRedis(ctx, rdb))

	now := time.Now().Unix()
	require.Equal(t, Phase1Pass, cache.Phase1Check(newIP, now, nil))

	addedAt := time.Now()
	require.NoError(t, rdb.SAdd(ctx, redisKeyBlacklistManual, newIP).Err())

	var blockedWithin time.Duration
	deadline := addedAt.Add(edgeBlacklistSyncInterval)
	var metrics Metrics

	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		require.NoError(t, cache.SyncFromRedis(ctx, rdb))
		now = time.Now().Unix()
		if cache.Phase1Check(newIP, now, &metrics) == Phase1Blocked403 {
			blockedWithin = time.Since(addedAt)
			break
		}
	}

	require.NotZero(t, blockedWithin, "edge must block IP within %s sync window", edgeBlacklistSyncInterval)
	assert.LessOrEqual(t, blockedWithin, edgeBlacklistSyncInterval)
	assert.Equal(t, int64(0), metrics.BodyRead, "blacklist block must occur before body read")
	assert.GreaterOrEqual(t, metrics.BlockedIP, int64(1))

	seconds := int(blockedWithin.Round(time.Second) / time.Second)
	if seconds < 1 {
		seconds = 1
	}

	faultproof.Log(t, "edge_blacklist_propagation", map[string]string{
		"blocked_within_seconds": strconv.Itoa(seconds),
		"body_read_total":        strconv.FormatInt(metrics.BodyRead, 10),
		"sync_interval_sec":      strconv.Itoa(int(edgeBlacklistSyncInterval / time.Second)),
		"harness":                "go_perimeter_mirror",
	})
}

func TestFault_EdgeFraudBlacklistPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("edge fault integration test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	const fraudIP = "203.0.113.99"
	cache := NewBlacklistCache(defaultStaleSec)
	require.NoError(t, cache.SyncFromRedis(ctx, rdb))

	now := time.Now().Unix()
	require.Equal(t, Phase1Pass, cache.Phase1Check(fraudIP, now, nil))

	addedAt := time.Now()
	require.NoError(t, rdb.SAdd(ctx, redisKeyBlacklistFraud, fraudIP).Err())

	var blockedWithin time.Duration
	deadline := addedAt.Add(edgeBlacklistSyncInterval)
	var metrics Metrics

	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		require.NoError(t, cache.SyncFromRedis(ctx, rdb))
		now = time.Now().Unix()
		if cache.Phase1Check(fraudIP, now, &metrics) == Phase1Blocked403 {
			blockedWithin = time.Since(addedAt)
			break
		}
	}

	require.NotZero(t, blockedWithin, "edge must block fraud IP within %s sync window", edgeBlacklistSyncInterval)
	assert.Equal(t, int64(0), metrics.BodyRead)

	faultproof.Log(t, "edge_fraud_blacklist_propagation", map[string]string{
		"source":            "blacklist:fraud",
		"blocked_within_ms": strconv.FormatInt(blockedWithin.Milliseconds(), 10),
		"harness":           "go_perimeter_mirror",
	})
}

func TestFault_ASNWhitelistBypass(t *testing.T) {
	if testing.Short() {
		t.Skip("edge fault integration test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	const blockedIP = "198.51.100.88"
	require.NoError(t, rdb.SAdd(ctx, redisKeyBlacklistManual, blockedIP).Err())

	cache := NewBlacklistCache(defaultStaleSec)
	cache.SetASNWhitelist(NewASNWhitelist("15169", ""))
	require.NoError(t, cache.SyncFromRedis(ctx, rdb))

	now := time.Now().Unix()
	var metrics Metrics

	outcome := cache.Phase1CheckASN(blockedIP, "15169", now, &metrics)
	assert.Equal(t, Phase1Pass, outcome)
	assert.Equal(t, int64(1), metrics.Phase1Pass)
	assert.Equal(t, int64(0), metrics.BlockedIP)
	assert.Equal(t, int64(0), metrics.BodyRead)

	faultproof.Log(t, "edge_asn_whitelist_bypass", map[string]string{
		"asn":                 "15169",
		"blocked_ip_bypassed": "true",
		"body_read_total":     "0",
		"harness":             "go_perimeter_mirror",
	})
}

func TestFault_EdgeBlacklistStale503(t *testing.T) {
	if testing.Short() {
		t.Skip("edge fault integration test")
	}

	cache := NewBlacklistCache(defaultStaleSec)
	var metrics Metrics

	outcome := cache.Phase1Check("198.51.100.1", time.Now().Unix(), &metrics)
	assert.Equal(t, Phase1Stale503, outcome)
	assert.Equal(t, int64(1), metrics.BlacklistStale)
	assert.Equal(t, int64(0), metrics.BodyRead)

	faultproof.Log(t, "edge_blacklist_staleness", map[string]string{
		"ttl_respected": "true",
		"stale_503":     "true",
		"harness":       "go_perimeter_mirror",
	})
}
