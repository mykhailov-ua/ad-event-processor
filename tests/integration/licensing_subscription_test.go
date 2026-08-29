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

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Licensing(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: licensing subscription (run make test-integration)")
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
			GraceDays:  7,
		}
		state := licensing.DetermineState(claims, time.Now(), false)
		assert.Equal(t, licensing.StateGrace, state)
	})

	t.Run("FileLicenseWatcher", func(t *testing.T) {
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
			Issuer:       "ad-event-processor-license",
			Subject:      uuid.NewString(),
			DeploymentID: uuid.NewString(),
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

		require.NoError(t, os.WriteFile(tempFile, []byte(token), 0o644))

		t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "file")
		t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", tempFile)

		watcher := licensing.NewLicenseWatcher(dbPool, rdb, pub)
		require.NoError(t, watcher.Start(ctx))

		time.Sleep(150 * time.Millisecond)

		state, loadedClaims := watcher.GetState()
		assert.Equal(t, licensing.StateActive, state)
		require.NotNil(t, loadedClaims)

		var dbState string
		err = dbPool.QueryRow(ctx, "SELECT state FROM billing.license_status LIMIT 1").Scan(&dbState)
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", dbState)
	})

	t.Run("HotPathEntitlementsFilter", func(t *testing.T) {
		customerID := uuid.New()
		campaignID := uuid.New()
		deploymentID := uuid.New()
		licenseID := uuid.New()

		_, err = dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, $3, $4)", customerID, "HotPath Cust", 500_000_000, "USD")
		require.NoError(t, err)

		_, err = dbPool.Exec(ctx, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id) VALUES ($1, $2, $3, $4, $5)",
			campaignID, "HotPath Campaign", 100_000_000, "ACTIVE", customerID)
		require.NoError(t, err)

		entitlementsJSON := `{"limits":{"max_active_campaigns":50,"max_rps":10000,"max_requests_per_day":2,"max_events_per_month":5000000,"max_regions":1,"max_api_keys":2,"max_export_chunk_bytes":1048576,"quota_reset_timezone":"UTC"},"features":{"rtb_live":true,"ml_fraud_boost":true,"multi_region":false,"slot_migration":false}}`
		_, err = dbPool.Exec(ctx, `
			INSERT INTO billing.license_status (deployment_id, license_id, plan_code, valid_until, state, entitlements_json, last_verified_at)
			VALUES ($1, $2, '', $3, 'ACTIVE', $4::jsonb, NOW())
		`, deploymentID, licenseID, time.Now().Add(24*time.Hour), entitlementsJSON)
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
