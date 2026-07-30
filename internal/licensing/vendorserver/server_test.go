package vendorserver_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"espx/internal/billing/db"
	"espx/internal/licensing"
	"espx/internal/licensing/vendorserver"
	"espx/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupVendorLicensePostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = nil
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	dir := testutil.BillingMigrationsDir()
	for _, name := range []string{"00003_vendor_licensing.sql", "00005_license_activations.sql"} {
		applyMigrationFile(t, pool, filepath.Join(dir, name))
	}
	return pool, cleanup
}

func applyMigrationFile(t *testing.T, pool *pgxpool.Pool, path string) {
	t.Helper()
	sqlBytes, err := os.ReadFile(path)
	require.NoError(t, err)
	sql := string(sqlBytes)
	parts := strings.Split(sql, "-- +goose Down")
	upPart := parts[0]
	upPart = strings.ReplaceAll(upPart, "-- +goose Up", "")
	upPart = strings.ReplaceAll(upPart, "-- +goose StatementBegin", "")
	upPart = strings.ReplaceAll(upPart, "-- +goose StatementEnd", "")
	_, err = pool.Exec(context.Background(), upPart)
	require.NoError(t, err, "migration %s", filepath.Base(path))
}

func TestVendorServer_cloneFingerprintDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := setupVendorLicensePostgres(t)
	defer cleanup()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	srv := vendorserver.New(pool, priv)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	queries := db.New(pool)
	limits, _ := json.Marshal(licensing.Limits{MaxRPS: 1000})
	features, _ := json.Marshal(licensing.FeatureSet{})
	_, err = queries.InsertVendorLicense(ctx, db.InsertVendorLicenseParams{
		LicenseKey:     "test-lic-clone",
		CustomerName:   "Acme",
		PlanCode:       "ingest_pro",
		ValidFrom:      pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		ValidUntil:     pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
		GraceDays:      7,
		LimitsJson:     limits,
		FeaturesJson:   features,
		SupportTier:    "standard",
		Revoked:        false,
		MaxActivations: 1,
	})
	require.NoError(t, err)

	depA := uuid.NewString()
	bodyA, _ := json.Marshal(map[string]string{
		"license_key":   "test-lic-clone",
		"deployment_id": depA,
		"fingerprint":   "fp-primary",
	})
	respA, err := http.Post(server.URL+"/v1/activate", "application/json", bytes.NewReader(bodyA))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, respA.StatusCode)
	_ = respA.Body.Close()

	depB := uuid.NewString()
	bodyB, _ := json.Marshal(map[string]string{
		"license_key":   "test-lic-clone",
		"deployment_id": depB,
		"fingerprint":   "fp-clone",
	})
	respB, err := http.Post(server.URL+"/v1/activate", "application/json", bytes.NewReader(bodyB))
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, respB.StatusCode)
	_ = respB.Body.Close()

	var queued int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM vendor.license_revoke_queue
		WHERE license_key = $1 AND reason = 'same_key_many_fingerprints' AND processed_at IS NULL
	`, "test-lic-clone").Scan(&queued)
	require.NoError(t, err)
	require.Equal(t, 0, queued)
}

func TestVendorServer_heartbeatJWTWithin72h(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := setupVendorLicensePostgres(t)
	defer cleanup()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	srv := vendorserver.New(pool, priv)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	queries := db.New(pool)
	limits, _ := json.Marshal(licensing.Limits{MaxRPS: 500})
	features, _ := json.Marshal(licensing.FeatureSet{})
	_, err = queries.InsertVendorLicense(ctx, db.InsertVendorLicenseParams{
		LicenseKey:     "test-lic-ttl",
		CustomerName:   "Buyer",
		PlanCode:       "ingest_pro",
		ValidFrom:      pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		ValidUntil:     pgtype.Timestamptz{Time: time.Now().Add(365 * 24 * time.Hour), Valid: true},
		GraceDays:      7,
		LimitsJson:     limits,
		FeaturesJson:   features,
		SupportTier:    "standard",
		Revoked:        false,
		MaxActivations: 1,
	})
	require.NoError(t, err)

	depID := uuid.NewString()
	body, _ := json.Marshal(map[string]string{
		"license_key":   "test-lic-ttl",
		"deployment_id": depID,
		"fingerprint":   "fp-ttl",
	})
	resp, err := http.Post(server.URL+"/v1/heartbeat", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	_ = resp.Body.Close()

	resp, err = http.Post(server.URL+"/v1/activate", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var res struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
	_ = resp.Body.Close()

	claims, err := licensing.VerifyJWT(res.Token, pub)
	require.NoError(t, err)
	require.True(t, claims.ValidUntil.Before(time.Now().Add(licensing.HeartbeatJWTMaxTTL+time.Minute)))
	require.True(t, claims.ValidUntil.After(time.Now().Add(licensing.HeartbeatJWTMaxTTL-time.Minute)))
}

func TestFault_LicenseHeartbeatReplayDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	ctx := context.Background()
	pool, cleanup := setupVendorLicensePostgres(t)
	defer cleanup()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	srv := vendorserver.New(pool, priv)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	queries := db.New(pool)
	limits, _ := json.Marshal(licensing.Limits{MaxRPS: 100})
	features, _ := json.Marshal(licensing.FeatureSet{})
	_, err = queries.InsertVendorLicense(ctx, db.InsertVendorLicenseParams{
		LicenseKey:     "fault-lic-replay",
		CustomerName:   "Fault",
		PlanCode:       "ingest_pro",
		ValidFrom:      pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		ValidUntil:     pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
		GraceDays:      7,
		LimitsJson:     limits,
		FeaturesJson:   features,
		SupportTier:    "standard",
		Revoked:        false,
		MaxActivations: 1,
	})
	require.NoError(t, err)

	depID := uuid.NewString()
	activateBody, _ := json.Marshal(map[string]string{
		"license_key":   "fault-lic-replay",
		"deployment_id": depID,
		"fingerprint":   "fp-bound",
	})
	resp, err := http.Post(server.URL+"/v1/activate", "application/json", bytes.NewReader(activateBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	replayBody, _ := json.Marshal(map[string]string{
		"license_key":   "fault-lic-replay",
		"deployment_id": depID,
		"fingerprint":   "fp-replay-clone",
	})
	resp, err = http.Post(server.URL+"/v1/heartbeat", "application/json", bytes.NewReader(replayBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	_ = resp.Body.Close()

	t.Log("fault_proof fault=license_heartbeat_replay subsystem=licensing denied=true baseline_ok=true")
}
