package licensing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/pkg/naming"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_LicenseServerUnreachableUsesLastKnownGood(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "license.jwt")

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		Plan:         "growth",
		VolumeBand:   entitlements.VolumeBandMedium,
		ValidFrom:    time.Now().Add(-24 * time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
		Limits:       entitlements.Limits{MaxRPS: 1000},
		Features:     entitlements.FeatureSet{OpenRTBEngine: true},
	}
	token := signFaultJWT(t, priv, claims)
	require.NoError(t, os.WriteFile(tokenPath, []byte(token), 0o640))

	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_MODE"), "online")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), tokenPath)
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_SERVER"), "http://127.0.0.1:1")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_KEY"), "fault-key")

	w := NewLicenseWatcher(nil, nil, pub)

	_, hbErr := w.performOnlineHeartbeat(context.Background())
	require.Error(t, hbErr)

	tokenStr, err := w.readLocalFile()
	require.NoError(t, err)
	loaded, err := VerifyJWT(tokenStr, pub)
	require.NoError(t, err)

	now := time.Now()
	state := entitlements.DetermineEffectiveState(loaded, now, false, now, true, w.policy)
	assert.Equal(t, entitlements.StateOfflineWarn, state)
	assert.Equal(t, entitlements.VolumeBandMedium, entitlements.ParseVolumeBand(string(loaded.VolumeBand)))

	t.Log("fault_proof fault=license_server_unreachable_last_known_good subsystem=licensing state=" + string(state))
}

func signFaultJWT(t *testing.T, priv ed25519.PrivateKey, claims entitlements.LicenseClaims) string {
	t.Helper()
	headerBytes, err := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "2026-01"})
	require.NoError(t, err)
	claimsBytes, err := json.Marshal(claims)
	require.NoError(t, err)
	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	sig := ed25519.Sign(priv, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}
