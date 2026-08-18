package controlplane

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/stretchr/testify/require"
)

func TestRevokeQueue_processesPendingRow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()
	database.ApplyLedgerMigrations(t, pool)

	ctx := context.Background()
	licenseKey := "550e8400-e29b-41d4-a716-446655440000"
	validUntil := time.Now().UTC().Add(24 * time.Hour)

	_, err := pool.Exec(ctx, `
		INSERT INTO vendor.licenses (
			license_key, customer_name, plan_code, valid_from, valid_until,
			grace_days, limits_json, features_json, support_tier, revoked
		) VALUES ($1, 'Test', 'pilot', NOW(), $2, 7, '{}', '{}', 'standard', false)`,
		licenseKey, validUntil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO vendor.license_revoke_queue (license_key, reason, detail_json)
		VALUES ($1, 'activation_abuse', '{"fingerprints":2}')`, licenseKey)
	require.NoError(t, err)

	reloaded := false
	worker := NewLicenseRevokeQueueWorker(pool, time.Minute)
	worker.reload = func(context.Context) error {
		reloaded = true
		return nil
	}
	worker.currentKey = func() string { return licenseKey }

	n, err := worker.ProcessOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.True(t, reloaded)

	var revoked bool
	var processed bool
	err = pool.QueryRow(ctx, `
		SELECT l.revoked, (q.processed_at IS NOT NULL)
		FROM vendor.licenses l
		JOIN vendor.license_revoke_queue q ON q.license_key = l.license_key
		WHERE l.license_key = $1`, licenseKey).Scan(&revoked, &processed)
	require.NoError(t, err)
	require.True(t, revoked)
	require.True(t, processed)
}

func TestRevokeQueue_skipsReloadForOtherLicense(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()
	database.ApplyLedgerMigrations(t, pool)

	ctx := context.Background()
	licenseKey := "550e8400-e29b-41d4-a716-446655440001"
	validUntil := time.Now().UTC().Add(24 * time.Hour)

	_, err := pool.Exec(ctx, `
		INSERT INTO vendor.licenses (
			license_key, customer_name, plan_code, valid_from, valid_until,
			grace_days, limits_json, features_json, support_tier, revoked
		) VALUES ($1, 'Other', 'pilot', NOW(), $2, 7, '{}', '{}', 'standard', false)`,
		licenseKey, validUntil)
	require.NoError(t, err)

	detail, err := json.Marshal(map[string]int{"fingerprints": 3})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO vendor.license_revoke_queue (license_key, reason, detail_json)
		VALUES ($1, 'activation_abuse', $2)`, licenseKey, detail)
	require.NoError(t, err)

	reloaded := false
	worker := NewLicenseRevokeQueueWorker(pool, time.Minute)
	worker.reload = func(context.Context) error {
		reloaded = true
		return nil
	}
	worker.currentKey = func() string { return "different-license-key" }

	n, err := worker.ProcessOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.False(t, reloaded)
}
