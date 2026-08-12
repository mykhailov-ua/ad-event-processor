package licensing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"github.com/bidshard/ad-event-processor/pkg/naming"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseWatcher_offlineGraceBlocksIngest(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "license.jwt")

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		Plan:         "growth",
		ValidFrom:    time.Now().Add(-24 * time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
	}
	token := signFaultJWT(t, priv, claims)
	require.NoError(t, os.WriteFile(tokenPath, []byte(token), 0o640))

	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_MODE"), "online")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), tokenPath)
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_SERVER"), "http://127.0.0.1:1")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_KEY"), "fault-key")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_OFFLINE_GRACE_DAYS"), "14")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_RENEW_BEFORE_DAYS"), "7")

	w := NewLicenseWatcher(nil, nil, pub)
	w.policy = HeartbeatPolicy{OfflineGraceDays: 14, RenewBeforeDays: 7}
	w.mu.Lock()
	w.offlineSince = time.Now().Add(-15 * 24 * time.Hour)
	w.mu.Unlock()

	require.NoError(t, w.verifyAndReload(context.Background()))
	state, _ := w.GetState()
	assert.Equal(t, StateExpired, state)
	assert.False(t, IngestAllowed(state))
	t.Log("fault_proof fault=license_offline_grace_expired subsystem=licensing state=EXPIRED")
}
