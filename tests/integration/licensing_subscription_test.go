package integration_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	"espx/internal/billing"
	"espx/internal/domain"
	"espx/internal/ingestion"
	db "espx/internal/domain/db"
	"espx/internal/licensing"
	"espx/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_LicensingAndSubscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping licensing integration test")
	}

	ctx := context.Background()

	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	dbPool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	t.Run("EntitlementsMerging", func(t *testing.T) {
		dep := licensing.Entitlements{
			Limits: licensing.Limits{
				MaxRPS:             1000,
				MaxRequestsPerDay:  50000,
				MaxActiveCampaigns: 5,
			},
			Features: licensing.FeatureSet{
				RtbLive:      true,
				MlFraudBoost: true,
			},
		}

		cust := licensing.Entitlements{
			Limits: licensing.Limits{
				MaxRPS:             2000,
				MaxRequestsPerDay:  30000,
				MaxActiveCampaigns: 0,
			},
			Features: licensing.FeatureSet{
				RtbLive:      true,
				MlFraudBoost: false,
			},
		}

		eff := licensing.Effective(dep, cust)
		assert.Equal(t, uint64(1000), eff.Limits.MaxRPS)
		assert.Equal(t, uint64(30000), eff.Limits.MaxRequestsPerDay)
		assert.Equal(t, uint64(5), eff.Limits.MaxActiveCampaigns)
		assert.True(t, eff.Features.RtbLive)
		assert.False(t, eff.Features.MlFraudBoost)
	})

	t.Run("LicenseStateTransitions", func(t *testing.T) {
		claims := &licensing.LicenseClaims{
			ValidFrom:  time.Now().Add(-2 * time.Hour),
			ValidUntil: time.Now().Add(-1 * time.Hour),
			GraceDays:  3,
		}

		nowInGrace := time.Now()
		stateGrace := licensing.DetermineState(claims, nowInGrace, false)
		assert.Equal(t, licensing.StateGrace, stateGrace)

		nowExpired := time.Now().Add(5 * 24 * time.Hour)
		stateExpired := licensing.DetermineState(claims, nowExpired, false)
		assert.Equal(t, licensing.StateExpired, stateExpired)

		stateRevoked := licensing.DetermineState(claims, nowInGrace, true)
		assert.Equal(t, licensing.StateRevoked, stateRevoked)
	})

	t.Run("LicenseWatcherAndOutbox", func(t *testing.T) {
		tempFile := t.TempDir() + "/license.jwt"

		limits := licensing.Limits{
			MaxRPS:             500,
			MaxRequestsPerDay:  10000,
			MaxActiveCampaigns: 10,
		}
		feats := licensing.FeatureSet{
			RtbLive: true,
		}

		claims := licensing.LicenseClaims{
			Issuer:       "espx-license",
			Subject:      uuid.NewString(),
			DeploymentID: uuid.NewString(),
			Plan:         "growth",
			ValidFrom:    time.Now().Add(-24 * time.Hour),
			ValidUntil:   time.Now().Add(24 * time.Hour),
			GraceDays:    7,
			Limits:       limits,
			Features:     feats,
		}

		headerBytes, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "2026-01"})
		claimsBytes, _ := json.Marshal(claims)
		signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
		sig := ed25519.Sign(priv, []byte(signingInput))
		token := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

		require.NoError(t, os.WriteFile(tempFile, []byte(token), 0644))

		t.Setenv("ESPX_LICENSE_MODE", "file")
		t.Setenv("ESPX_LICENSE_PATH", tempFile)

		watcher := licensing.NewLicenseWatcher(dbPool, rdb, pub)
		require.NoError(t, watcher.Start(ctx))

		time.Sleep(150 * time.Millisecond)

		state, loadedClaims := watcher.GetState()
		assert.Equal(t, licensing.StateActive, state)
		require.NotNil(t, loadedClaims)
		assert.Equal(t, "growth", loadedClaims.Plan)

		var dbState string
		var dbPlan string
		err = dbPool.QueryRow(ctx, "SELECT state, plan_code FROM billing.license_status LIMIT 1").Scan(&dbState, &dbPlan)
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", dbState)
		assert.Equal(t, "growth", dbPlan)

		rPlan, err := rdb.HGet(ctx, "entitlement:deployment", "plan").Result()
		require.NoError(t, err)
		assert.Equal(t, "growth", rPlan)
	})

	t.Run("SubscriptionBillingAndOverage", func(t *testing.T) {
		customerID := uuid.New()
		_, err = dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, $3, $4)", customerID, "Subscription Cust", 95_000_000, "USD")
		require.NoError(t, err)

		limitsRaw := `{"max_active_campaigns": 50, "max_rps": 10000, "max_requests_per_day": 500000, "max_events_per_month": 10000, "max_regions": 1, "max_api_keys": 2, "max_export_chunk_bytes": 1048576, "quota_reset_timezone": "UTC"}`
		featuresRaw := `{"rtb_live": false, "ml_fraud_boost": false, "multi_region": false, "slot_migration": false}`
		_, err = dbPool.Exec(ctx, `
			INSERT INTO billing.subscription_plans (code, display_name, limits_json, features_json, base_fee_micro)
			VALUES ($1, $2, $3, $4, $5)
		`, "basic_test", "Basic Test", []byte(limitsRaw), []byte(featuresRaw), int64(10_000_000))
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO billing.customer_subscriptions (customer_id, plan_code, status, period_start)
			VALUES ($1, $2, $3, $4)
		`, customerID, "basic_test", "active", time.Now().Add(-2*time.Hour))
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, $2, 'TOPUP', $3)
		`, customerID, int64(100_000_000), "topup-test")
		require.NoError(t, err)

		monthStart := truncateMonthUTC(time.Now())
		_, err = dbPool.Exec(ctx, `
			INSERT INTO billing.usage_meters (customer_id, meter, period, value)
			VALUES ($1, $2, $3, $4)
		`, customerID, "events", monthStart, int64(12000))
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, $2, 'FEE', $3)
		`, customerID, int64(-5_000_000), "ad-spend-test")
		require.NoError(t, err)

		billingSvc := billing.NewService(dbPool)
		inv, err := billingSvc.GenerateInvoice(ctx, customerID, monthStart)
		require.NoError(t, err)

		assert.Equal(t, int64(15_100_000), inv.SubtotalMicro)
		assert.Equal(t, int64(16_194_750), inv.TotalMicro)

		var dbBalance int64
		err = dbPool.QueryRow(ctx, "SELECT balance FROM public.customers WHERE id = $1", customerID).Scan(&dbBalance)
		require.NoError(t, err)
		assert.Equal(t, int64(84_900_000), dbBalance)
	})

	t.Run("HotPathEntitlementsFilter", func(t *testing.T) {
		customerID := uuid.New()
		campaignID := uuid.New()

		_, err = dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, $3, $4)", customerID, "HotPath Cust", 500_000_000, "USD")
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id) VALUES ($1, $2, $3, $4, $5)",
			campaignID, "HotPath Campaign", 100_000_000, "ACTIVE", customerID)
		require.NoError(t, err)

		limitsRaw := `{"max_active_campaigns": 50, "max_rps": 10000, "max_requests_per_day": 2, "max_events_per_month": 5000000, "max_regions": 1, "max_api_keys": 2, "max_export_chunk_bytes": 1048576, "quota_reset_timezone": "UTC"}`
		featuresRaw := `{"rtb_live": true, "ml_fraud_boost": true, "multi_region": false, "slot_migration": false}`
		_, err = dbPool.Exec(ctx, `
			INSERT INTO billing.subscription_plans (code, display_name, limits_json, features_json, base_fee_micro)
			VALUES ($1, $2, $3, $4, $5)
		`, "hot_test", "Hot Test", []byte(limitsRaw), []byte(featuresRaw), int64(0))
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, `
			INSERT INTO billing.customer_subscriptions (customer_id, plan_code, status, period_start)
			VALUES ($1, $2, $3, $4)
		`, customerID, "hot_test", "active", time.Now().Add(-2*time.Hour))
		require.NoError(t, err)

		queries := db.New(dbPool)
		registry := ingestion.NewRegistry(queries)
		registry.SetPool(dbPool)

		_, err = registry.Sync(ctx)
		require.NoError(t, err)

		sharder := ingestion.NewStaticSlotSharder(1)
		filter := ingestion.NewEntitlementsFilter(registry, sharder, []redis.UniversalClient{rdb})

		evt := &domain.Event{
			CampaignID: campaignID,
			Type:       "impression",
		}

		err = filter.Check(ctx, evt)
		assert.NoError(t, err)

		err = filter.Check(ctx, evt)
		assert.NoError(t, err)

		err = filter.Check(ctx, evt)
		assert.ErrorIs(t, err, ingestion.ErrDailyQuotaExceeded)
	})
}

func truncateMonthUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
