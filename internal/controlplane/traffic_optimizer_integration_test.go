package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/trafficoptimizer"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flowLanderFixture struct {
	customerID uuid.UUID
	campaignID uuid.UUID
	flowID     uuid.UUID
	landerA    uuid.UUID
	landerB    uuid.UUID
}

func setupTrafficOptimizerIntegration(t *testing.T) (*pgxpool.Pool, redis.UniversalClient, driver.Conn, *Service) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration: traffic optimizer end-to-end")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	t.Cleanup(cleanupDB)
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)
	chConn, cleanupCH := setupClickHouseStatsTest(t)
	t.Cleanup(cleanupCH)
	require.NoError(t, migrate.ApplyClickHouseMigrations(context.Background(), chConn))

	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, nil)
	svc.SetClickHouse(chConn, database.ClickHouseQueryConfig{})
	t.Cleanup(func() { svc.Close() })
	return pool, redisClient, chConn, svc
}

func seedFlowLanderFixture(t *testing.T, ctx context.Context, svc *Service, idem string) flowLanderFixture {
	t.Helper()
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "TO "+idem, 500_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, testCampaignSpec(customerID, idem+" Camp", 100_000_000, idem+"-idem"))
	require.NoError(t, err)

	landerA := uuid.New()
	landerB := uuid.New()
	_, err = svc.GetPool().Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'LA', 'https://a.example/'), ($2, 'LB', 'https://b.example/')`, landerA, landerB)
	require.NoError(t, err)

	flowID := uuid.New()
	paths, err := json.Marshal([]flow.PathDTO{{
		Weight: 100,
		Landers: []flow.PathLanderRef{
			{LanderID: landerA, Weight: 50},
			{LanderID: landerB, Weight: 50},
		},
	}})
	require.NoError(t, err)
	_, err = svc.GetPool().Exec(ctx, `INSERT INTO flows (id, name, paths) VALUES ($1, $2, $3)`, flowID, idem+"-flow", paths)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCampaignFlow(ctx, campID, flowID))

	return flowLanderFixture{
		customerID: customerID,
		campaignID: campID,
		flowID:     flowID,
		landerA:    landerA,
		landerB:    landerB,
	}
}

func insertLanderBanditCH(
	t *testing.T,
	ctx context.Context,
	conn driver.Conn,
	campaignID, landerID uuid.UUID,
	clickCount int,
	spendMicroPerClick int64,
	conversionCount int,
	payoutPerConversion float64,
	at time.Time,
) {
	t.Helper()
	landerStr := landerID.String()
	for i := range clickCount {
		payload := fmt.Sprintf(`{"lander_id":"%s","spend_micro":"%d"}`, landerStr, spendMicroPerClick)
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO clicks (click_id, campaign_id, placement_id, ip_hash, ua_hash, payload, created_at)
			VALUES (?, ?, 'zone-1', unhex('00000000000000000000000000000000'), unhex('00000000000000000000000000000000'), ?, ?)`,
			fmt.Sprintf("clk-%s-%d", landerStr[:8], i), campaignID, payload, at.Add(time.Duration(i)*time.Millisecond),
		))
	}
	for i := range conversionCount {
		payload := fmt.Sprintf(`{"lander_id":"%s","payout":"%g"}`, landerStr, payoutPerConversion)
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO conversions (click_id, campaign_id, placement_id, ip_hash, ua_hash, payload, created_at)
			VALUES (?, ?, 'zone-1', unhex('00000000000000000000000000000000'), unhex('00000000000000000000000000000000'), ?, ?)`,
			fmt.Sprintf("conv-%s-%d", landerStr[:8], i), campaignID, payload, at.Add(time.Duration(i)*time.Millisecond),
		))
	}
}

func readLanderWeights(t *testing.T, ctx context.Context, svc *Service, flowID, landerA, landerB uuid.UUID) (int32, int32) {
	t.Helper()
	var pathsRaw []byte
	require.NoError(t, svc.GetPool().QueryRow(ctx, `SELECT paths FROM flows WHERE id = $1`, flowID).Scan(&pathsRaw))
	var paths []flow.PathDTO
	require.NoError(t, json.Unmarshal(pathsRaw, &paths))
	require.NotEmpty(t, paths)
	var weightA, weightB int32
	for _, l := range paths[0].Landers {
		switch l.LanderID {
		case landerA:
			weightA = l.Weight
		case landerB:
			weightB = l.Weight
		}
	}
	return weightA, weightB
}

func insertTrafficOptimizerRule(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fix flowLanderFixture, rule db.InsertTrafficOptimizerRuleParams) trafficoptimizer.Rule {
	t.Helper()
	rule.CustomerID = domain.ToUUID(fix.customerID)
	rule.CampaignID = domain.ToUUID(fix.campaignID)
	if len(rule.PresetParameters) == 0 {
		rule.PresetParameters = []byte("{}")
	}
	row, err := db.New(pool).InsertTrafficOptimizerRule(ctx, rule)
	require.NoError(t, err)
	out, err := trafficoptimizer.RuleFromRow(row)
	require.NoError(t, err)
	return out
}

func applyTrafficOptimizerRule(t *testing.T, ctx context.Context, pool *pgxpool.Pool, svc *Service, rule trafficoptimizer.Rule, windowEnd time.Time) bool {
	t.Helper()
	host := trafficOptimizerHost{svc: svc}
	var applied bool
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, ok, err := trafficoptimizer.ApplyRuleTx(ctx, tx, host, rule, windowEnd)
		if err != nil {
			return err
		}
		applied = ok
		return nil
	})
	require.NoError(t, err)
	return applied
}

func TestTrafficOptimizer_CR_EndToEnd(t *testing.T) {
	pool, redisClient, chConn, svc := setupTrafficOptimizerIntegration(t)
	ctx := context.Background()
	fix := seedFlowLanderFixture(t, ctx, svc, "cr")
	windowEnd := time.Now().UTC().Truncate(time.Second)
	eventAt := windowEnd.Add(-30 * time.Minute)

	insertLanderBanditCH(t, ctx, chConn, fix.campaignID, fix.landerA, 500, 0, 200, 0, eventAt)
	insertLanderBanditCH(t, ctx, chConn, fix.campaignID, fix.landerB, 500, 0, 5, 0, eventAt)

	var spendBefore int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, fix.campaignID).Scan(&spendBefore))

	rule := insertTrafficOptimizerRule(t, ctx, pool, fix, db.InsertTrafficOptimizerRuleParams{
		Name:                "CR lander shift",
		Scope:               trafficoptimizer.ScopeLander,
		Objective:           trafficoptimizer.ObjectiveCR,
		Algorithm:           trafficoptimizer.AlgorithmThompson,
		LookbackMinutes:     60,
		MinClicks:           100,
		EvalIntervalMinutes: 15,
		CooldownMinutes:     15,
		MaxWeightDeltaPct:   50,
		Enabled:             true,
	})

	var applied bool
	var weightA, weightB int32
	for attempt := range 30 {
		if applyTrafficOptimizerRule(t, ctx, pool, svc, rule, windowEnd.Add(time.Duration(attempt)*time.Second)) {
			applied = true
			weightA, weightB = readLanderWeights(t, ctx, svc, fix.flowID, fix.landerA, fix.landerB)
			if weightA > weightB {
				break
			}
		}
	}
	require.True(t, applied, "CR rule should apply weight shift from CH stats")
	assert.Greater(t, weightA, weightB, "higher CR lander must receive higher weight after Thompson apply")
	assert.NotEqual(t, int32(50), weightA)
	assert.NotEqual(t, int32(50), weightB)

	var spendAfter int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, fix.campaignID).Scan(&spendAfter))
	assert.Equal(t, spendBefore, spendAfter, "holdout: weight-only optimizer tick must not change current_spend")
	domain.AssertBudgetInvariant(t, ctx, pool, redisClient, fix.campaignID)
}

func TestTrafficOptimizer_ROI_EndToEnd(t *testing.T) {
	pool, redisClient, chConn, svc := setupTrafficOptimizerIntegration(t)
	ctx := context.Background()
	fix := seedFlowLanderFixture(t, ctx, svc, "roi")
	windowEnd := time.Now().UTC().Truncate(time.Second)
	eventAt := windowEnd.Add(-30 * time.Minute)

	insertLanderBanditCH(t, ctx, chConn, fix.campaignID, fix.landerA, 200, 10_000, 80, 5.0, eventAt)
	insertLanderBanditCH(t, ctx, chConn, fix.campaignID, fix.landerB, 200, 10_000, 80, 1.0, eventAt)

	var spendBefore int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, fix.campaignID).Scan(&spendBefore))

	rule := insertTrafficOptimizerRule(t, ctx, pool, fix, db.InsertTrafficOptimizerRuleParams{
		Name:                "ROI lander shift",
		Scope:               trafficoptimizer.ScopeLander,
		Objective:           trafficoptimizer.ObjectiveROI,
		Algorithm:           trafficoptimizer.AlgorithmProportional,
		LookbackMinutes:     60,
		MinClicks:           100,
		MinSpendMicro:       1_000_000,
		EvalIntervalMinutes: 15,
		CooldownMinutes:     15,
		MaxWeightDeltaPct:   50,
		Enabled:             true,
	})
	require.True(t, applyTrafficOptimizerRule(t, ctx, pool, svc, rule, windowEnd), "ROI rule should apply weight shift from CH stats")

	weightA, weightB := readLanderWeights(t, ctx, svc, fix.flowID, fix.landerA, fix.landerB)
	assert.Greater(t, weightA, weightB, "higher ROI lander must receive higher weight")
	assert.NotEqual(t, int32(50), weightA)
	assert.NotEqual(t, int32(50), weightB)

	var spendAfter int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, fix.campaignID).Scan(&spendAfter))
	assert.Equal(t, spendBefore, spendAfter, "holdout: weight-only optimizer tick must not change current_spend")
	domain.AssertBudgetInvariant(t, ctx, pool, redisClient, fix.campaignID)
}
