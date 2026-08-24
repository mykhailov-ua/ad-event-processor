package integration_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/ingestion"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func signLicenseToken(t *testing.T, priv ed25519.PrivateKey, claims licensing.LicenseClaims) string {
	t.Helper()
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	return token
}

func TestIntegration_LicenseProtection_emptyPGRowBlocksIngest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping license protection integration test")
	}
	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	_, err := pool.Exec(ctx, "DELETE FROM billing.license_status")
	require.NoError(t, err)

	registry := ingestion.NewRegistry(db.New(pool))
	registry.SetPool(pool)
	require.NoError(t, registry.SyncEntitlements(ctx))

	filter := ingestion.NewLicenseFilter(registry)
	err = filter.Check(ctx, &domain.Event{})
	require.ErrorIs(t, err, ingestion.ErrLicenseExpired)
}

func TestIntegration_LicenseProtection_hwidMismatchBlocksIngest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping license protection integration test")
	}
	ctx := context.Background()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
		HWIDHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Limits: licensing.Limits{
			MaxRPS:             1000,
			MaxRequestsPerDay:  100000,
			MaxActiveCampaigns: 10,
		},
	}
	claims.Bind.Mode = "hard"
	require.NoError(t, os.WriteFile(path, []byte(signLicenseToken(t, priv, claims)), 0o600))

	registry := ingestion.NewRegistry(nil)
	registry.StartLicenseRecheck(ctx, ingestion.RegistryLicenseConfig{
		Required: true,
		Path:     path,
		PubKey:   pub,
		Interval: time.Hour,
	})

	filter := ingestion.NewLicenseFilter(registry)
	err = filter.Check(ctx, &domain.Event{})
	require.ErrorIs(t, err, ingestion.ErrLicenseExpired)
}

func TestIntegration_LicenseProtection_hwidMatchAllowsIngest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping license protection integration test")
	}
	ctx := context.Background()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tel := licensing.HWIDTelemetry{
		DMIUUID:  "integration-dmi",
		DiskID:   "integration-disk",
		MAC:      "de:ad:be:ef:00:01",
		CPUModel: "Integration CPU",
		CPUCores: 4,
	}
	restore := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry { return tel })
	defer restore()

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
		HWIDHash:     licensing.HashHWIDFromTelemetry(tel),
		Limits: licensing.Limits{
			MaxRPS:             1000,
			MaxRequestsPerDay:  100000,
			MaxActiveCampaigns: 10,
		},
	}
	claims.Bind.Mode = "hard"
	require.NoError(t, os.WriteFile(path, []byte(signLicenseToken(t, priv, claims)), 0o600))

	registry := ingestion.NewRegistry(nil)
	registry.StartLicenseRecheck(ctx, ingestion.RegistryLicenseConfig{
		Required: true,
		Path:     path,
		PubKey:   pub,
		Interval: time.Hour,
	})

	filter := ingestion.NewLicenseFilter(registry)
	require.NoError(t, filter.Check(ctx, &domain.Event{}))
}

func TestIntegration_LicenseProtection_validJWTAllowsIngest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping license protection integration test")
	}
	ctx := context.Background()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
		Limits: licensing.Limits{
			MaxRPS:             1000,
			MaxRequestsPerDay:  100000,
			MaxActiveCampaigns: 10,
		},
	}
	require.NoError(t, os.WriteFile(path, []byte(signLicenseToken(t, priv, claims)), 0o600))

	registry := ingestion.NewRegistry(nil)
	registry.StartLicenseRecheck(ctx, ingestion.RegistryLicenseConfig{
		Required: true,
		Path:     path,
		PubKey:   pub,
		Interval: time.Hour,
	})

	filter := ingestion.NewLicenseFilter(registry)
	require.NoError(t, filter.Check(ctx, &domain.Event{}))
}

func TestIntegration_LicenseProtection_fakePGRowWithoutJWTBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping license protection integration test")
	}
	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	expiredClaims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-72 * time.Hour),
		ValidUntil:   time.Now().Add(-48 * time.Hour),
		GraceDays:    0,
	}
	require.NoError(t, os.WriteFile(path, []byte(signLicenseToken(t, priv, expiredClaims)), 0o600))

	entitlementsJSON := `{"limits":{"max_active_campaigns":99999,"max_rps":999999,"max_requests_per_day":999999999},"features":{"rtb_live":true}}`
	_, err = pool.Exec(ctx, `
		INSERT INTO billing.license_status (deployment_id, license_id, plan_code, valid_until, state, entitlements_json, last_verified_at)
		VALUES ($1, $2, 'pilot', $3, 'ACTIVE', $4::jsonb, NOW())
	`, uuid.New(), uuid.New(), time.Now().Add(365*24*time.Hour), entitlementsJSON)
	require.NoError(t, err)

	registry := ingestion.NewRegistry(db.New(pool))
	registry.SetPool(pool)
	registry.StartLicenseRecheck(ctx, ingestion.RegistryLicenseConfig{
		Required: true,
		Path:     path,
		PubKey:   pub,
		Interval: time.Hour,
	})

	filter := ingestion.NewLicenseFilter(registry)
	err = filter.Check(ctx, &domain.Event{})
	require.ErrorIs(t, err, ingestion.ErrLicenseExpired)
}
