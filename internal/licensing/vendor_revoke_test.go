package licensing_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/pkg/naming"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLicenseWatcher_vendorDBRevoke(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()
	database.ApplyLedgerMigrations(t, pool)
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "license.jwt")

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	depID := uuid.NewString()
	licenseKey := depID
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      licenseKey,
		DeploymentID: depID,
		Plan:         "pilot",
		ValidFrom:    time.Now().Add(-24 * time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenPath, []byte(token), 0o640))

	ctx := context.Background()
	_, err = pool.Exec(ctx, `
		INSERT INTO vendor.licenses (
			license_key, customer_name, plan_code, valid_from, valid_until,
			grace_days, limits_json, features_json, support_tier, revoked
		) VALUES ($1, 'Test', 'pilot', NOW(), $2, 7, '{}', '{}', 'standard', true)`,
		licenseKey, claims.ValidUntil)
	require.NoError(t, err)

	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_MODE"), "file")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), tokenPath)

	w := licensing.NewLicenseWatcher(pool, redisClient, pub)
	require.NoError(t, w.Reload(ctx))

	state, _ := w.GetState()
	require.Equal(t, entitlements.StateRevoked, state)
}
