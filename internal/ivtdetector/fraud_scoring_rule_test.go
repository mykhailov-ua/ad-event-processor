package ivtdetector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"espx/internal/database"
	"espx/internal/fraud"
	"espx/pkg/piihash"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFraudModelPath(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("FRAUD_TEST_MODEL"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		t.Skipf("FRAUD_TEST_MODEL not found: %s", path)
	}
	candidates := []string{
		filepath.Join("..", "..", "var", "fraudscore", "artifacts", "model.txt"),
		filepath.Join("..", "fraud", "testdata", "model.txt"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("fraud model not found; run make fraudtrain-check locally")
	return ""
}

func TestFraudScoringRule_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, cleanup := setupClickHouseTest(t)
	defer cleanup()

	ctx := context.Background()
	ensureFraudScoringShadowTables(t, conn)

	campaignID := uuid.New()
	now := time.Now().Truncate(time.Minute)

	insertQuery := `
		INSERT INTO ad_event_processor.ml_features_1m
		(window_start, ip_hash, campaign_id, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	h := testPIIHasher()
	err := conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP("1.2.3.4")), campaignID, uint64(10), uint64(2), int64(1000000), int64(5000000), uint64(1), uint64(1))
	require.NoError(t, err)

	err = conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP("5.6.7.8")), campaignID, uint64(100), uint64(10), int64(10000000), int64(50000000), uint64(5), uint64(2))
	require.NoError(t, err)

	scorer, err := fraud.NewLGBMScorer(testFraudModelPath(t))
	require.NoError(t, err)

	rule := NewFraudScoringRule(database.NewCHQuery(conn, database.CHQueryConfig{}), conn, nil, scorer, 100)
	assert.Equal(t, "fraud_scoring_shadow", rule.Name())

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	assert.Len(t, candidates, 1)
	assert.Equal(t, ipHashHex("1.2.3.4"), candidates[0].IP)
	assert.Equal(t, "boost", candidates[0].Action)
	assert.Equal(t, int32(52), candidates[0].Boost)

	var count uint64
	err = conn.QueryRow(ctx, "SELECT count() FROM ad_event_processor.ml_shadow_scores").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	var score1, score2 float64
	err = conn.QueryRow(ctx, "SELECT score FROM ad_event_processor.ml_shadow_scores WHERE ip_hash = ? LIMIT 1", piihash.FixedString16(h.HashIP("1.2.3.4"))).Scan(&score1)
	require.NoError(t, err)
	assert.InDelta(t, 0.52497, score1, 1e-4)

	err = conn.QueryRow(ctx, "SELECT score FROM ad_event_processor.ml_shadow_scores WHERE ip_hash = ? LIMIT 1", piihash.FixedString16(h.HashIP("5.6.7.8"))).Scan(&score2)
	require.NoError(t, err)
	assert.InDelta(t, 0.71094, score2, 1e-4)

	assert.Greater(t, score2, score1, "fraud-like IP must score higher than control IP")
}

func TestFraudScoringRule_FraudScoresHigherThanControl(t *testing.T) {
	if testing.Short() {
		t.Skip("clickhouse integration test")
	}

	conn, cleanup := setupClickHouseTest(t)
	defer cleanup()

	ctx := context.Background()
	ensureFraudScoringShadowTables(t, conn)

	campaignID := uuid.New()
	now := time.Now().Truncate(time.Minute)
	insertQuery := `
		INSERT INTO ad_event_processor.ml_features_1m
		(window_start, ip_hash, campaign_id, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	h := testPIIHasher()
	controlIP := "203.0.113.10"
	fraudIP := "203.0.113.20"

	require.NoError(t, conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP(controlIP)), campaignID, uint64(20), uint64(1), int64(1000000), int64(5000000), uint64(1), uint64(1)))
	require.NoError(t, conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP(fraudIP)), campaignID, uint64(200), uint64(50), int64(10000000), int64(50000000), uint64(20), uint64(1)))

	scorer, err := fraud.NewLGBMScorer(testFraudModelPath(t))
	require.NoError(t, err)

	_, err = NewFraudScoringRule(database.NewCHQuery(conn, database.CHQueryConfig{}), conn, nil, scorer, 100).Find(ctx)
	require.NoError(t, err)

	var controlScore, fraudScore float64
	require.NoError(t, conn.QueryRow(ctx, "SELECT score FROM ad_event_processor.ml_shadow_scores WHERE ip_hash = ? LIMIT 1", piihash.FixedString16(h.HashIP(controlIP))).Scan(&controlScore))
	require.NoError(t, conn.QueryRow(ctx, "SELECT score FROM ad_event_processor.ml_shadow_scores WHERE ip_hash = ? LIMIT 1", piihash.FixedString16(h.HashIP(fraudIP))).Scan(&fraudScore))

	assert.Greater(t, fraudScore, controlScore, "seeded fraud IP should outrank control IP")
}

type mockScorer struct {
	scores []float64
}

func (m *mockScorer) Name() string {
	return "mock-scorer"
}

func (m *mockScorer) ScoreBatch(ctx context.Context, rows []fraud.FeatureRow) ([]float64, error) {
	return m.scores, nil
}

func (m *mockScorer) Dims() int {
	return 8
}

func TestFraudScoringRule_WithCampaignThresholds(t *testing.T) {
	if testing.Short() {
		t.Skip("clickhouse integration test")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	ensureFraudScoringShadowTables(t, conn)

	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, budget_limit, fraud_threshold_pass, fraud_threshold_suspect, fraud_threshold_block, ghost_ivt_enabled)
		VALUES ($1, 'Test Campaign', 'ACTIVE', 1000000000, 20, 50, 90, true)
	`, campaignID)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Minute)
	insertQuery := `
		INSERT INTO ad_event_processor.ml_features_1m
		(window_start, ip_hash, campaign_id, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	h := testPIIHasher()
	require.NoError(t, conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP("1.1.1.1")), campaignID, uint64(10), uint64(2), int64(1000000), int64(5000000), uint64(1), uint64(1)))
	require.NoError(t, conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP("2.2.2.2")), campaignID, uint64(50), uint64(5), int64(5000000), int64(10000000), uint64(3), uint64(1)))
	require.NoError(t, conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP("3.3.3.3")), campaignID, uint64(100), uint64(10), int64(10000000), int64(20000000), uint64(5), uint64(2)))
	require.NoError(t, conn.Exec(ctx, insertQuery, now, piihash.FixedString16(h.HashIP("4.4.4.4")), campaignID, uint64(200), uint64(30), int64(20000000), int64(30000000), uint64(10), uint64(3)))

	scorer := &mockScorer{
		scores: []float64{0.1, 0.4, 0.75, 0.95},
	}

	rule := NewFraudScoringRule(database.NewCHQuery(conn, database.CHQueryConfig{}), conn, pool, scorer, 100)

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)

	candidateMap := make(map[string]SuspiciousIP)
	for _, c := range candidates {
		candidateMap[c.IP] = c
	}

	_, exists := candidateMap[ipHashHex("4.4.4.4")]
	assert.False(t, exists)

	c3, exists := candidateMap[ipHashHex("3.3.3.3")]
	require.True(t, exists)
	assert.Equal(t, "boost", c3.Action)
	assert.Equal(t, int32(40), c3.Boost)

	c2, exists := candidateMap[ipHashHex("2.2.2.2")]
	require.True(t, exists)
	assert.Equal(t, "ghost", c2.Action)

	c1, exists := candidateMap[ipHashHex("1.1.1.1")]
	require.True(t, exists)
	assert.Equal(t, "blacklist", c1.Action)
}
