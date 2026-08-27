package ingestion

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegistry_licenseRecheck_clockSkewBlocksIngest(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: registry skew (run make test-integration)")
	}

	licensing.ResetSkewWatchForTest()
	t.Cleanup(licensing.ResetSkewWatchForTest)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

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
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_SKEW_WATCH", "1")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	restoreClock := licensing.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})
	t.Cleanup(restoreClock)

	registry := NewRegistry(nil)
	registry.StartLicenseRecheck(ctx, RegistryLicenseConfig{
		Required: true,
		Path:     path,
		PubKey:   pub,
		Interval: time.Hour,
	})

	filter := NewLicenseFilter(registry)
	require.NoError(t, filter.Check(ctx, &domain.Event{}))

	mono = 2 * time.Hour
	wall = wall.Add(-30 * 24 * time.Hour)
	registry.recheckLicenseFile(context.Background(), licensing.FileLicenseRecheckConfig{
		Path:   path,
		PubKey: pub,
	})

	err = filter.Check(ctx, &domain.Event{})
	require.ErrorIs(t, err, ErrLicenseExpired)
}
