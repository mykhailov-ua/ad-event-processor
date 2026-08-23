package fraud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type poolBlacklistBlocker struct {
	pool *pgxpool.Pool
}

func (blocker *poolBlacklistBlocker) BlockIP(ctx context.Context, ip string) error {
	if edge.IsProtected(ip) {
		return fmt.Errorf("IP %s is protected by allowlist", ip)
	}
	_, err := blocker.pool.Exec(ctx, `
		INSERT INTO ip_blacklist (ip, reason)
		VALUES ($1, 'fraud')
		ON CONFLICT (ip) DO NOTHING
	`, ip)
	return err
}

func (blocker *poolBlacklistBlocker) EnqueueFraudThreat(context.Context, string, string, string, float64, int32, int64) error {
	return fmt.Errorf("not implemented")
}

func (blocker *poolBlacklistBlocker) EnqueueFraudThreatBatch(context.Context, []FraudThreatEnqueueItem) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestFault_ivtIntervalAutoblock(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	protectedIP := "8.8.8.8"
	botIP := "203.0.113.50"
	seedIntervalBotClicks(t, conn, protectedIP, "timer-bot-protected", 35, time.Second)
	seedIntervalBotClicks(t, conn, botIP, "timer-bot-open", 35, time.Second)

	rule := &intervalBotnetRule{
		q: database.NewCHQuery(conn, database.CHQueryConfig{}),
		cfg: AnalyzerConfig{
			Window:               time.Hour,
			IntervalMinIntervals: 30,
			IntervalMaxVariance:  0.005,
		},
	}
	candidates, err := rule.Find(ctx)
	require.NoError(t, err)

	foundProtected := false
	foundBot := false
	for _, candidate := range candidates {
		switch candidate.IP {
		case ipHashHex(protectedIP):
			foundProtected = true
		case ipHashHex(botIP):
			foundBot = true
		}
	}
	require.True(t, foundProtected, "expected protected timer bot in candidates")
	require.True(t, foundBot, "expected open timer bot in candidates")

	blocker := &poolBlacklistBlocker{pool: pool}

	err = blocker.BlockIP(ctx, protectedIP)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "protected by allowlist")

	detector := NewDetector(
		stubFinder{ips: []SuspiciousIP{
			{IP: botIP, Reason: intervalBotReason, Score: 0.0},
		}},
		NewIdempotencyStore(pool),
		blocker,
		pool,
		DetectorConfig{OutboxPendingLimit: 0},
	)

	result, err := detector.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Enqueued)

	var protectedBlacklist int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ip_blacklist WHERE ip = $1", protectedIP).Scan(&protectedBlacklist)
	require.NoError(t, err)
	assert.Equal(t, 0, protectedBlacklist)

	var botBlacklist int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ip_blacklist WHERE ip = $1", botIP).Scan(&botBlacklist)
	require.NoError(t, err)
	assert.Equal(t, 1, botBlacklist)

	faultproof.Log(t, "ivt_interval_autoblock", map[string]string{
		"subsystem":           "ivt_detector",
		"allowlist_respected": "true",
		"protected_ip":        protectedIP,
		"blocked_ip":          botIP,
	})
}
