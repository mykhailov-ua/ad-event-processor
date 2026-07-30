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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_LicenseServerUnreachableUsesLastKnownGood(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "license.jwt")

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := LicenseClaims{
		Issuer:       "espx-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		Plan:         "growth",
		VolumeBand:   VolumeBandMedium,
		ValidFrom:    time.Now().Add(-24 * time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
		Limits:       Limits{MaxRPS: 1000},
		Features:     FeatureSet{OpenRTBEngine: true},
	}
	token := signFaultJWT(t, priv, claims)
	require.NoError(t, os.WriteFile(tokenPath, []byte(token), 0o640))

	t.Setenv("ESPX_LICENSE_MODE", "online")
	t.Setenv("ESPX_LICENSE_PATH", tokenPath)
	t.Setenv("ESPX_LICENSE_SERVER", "http://127.0.0.1:1")
	t.Setenv("ESPX_LICENSE_KEY", "fault-key")

	w := NewLicenseWatcher(nil, nil, pub)

	_, hbErr := w.performOnlineHeartbeat(context.Background())
	require.Error(t, hbErr)

	tokenStr, err := w.readLocalFile()
	require.NoError(t, err)
	loaded, err := VerifyJWT(tokenStr, pub)
	require.NoError(t, err)
	state := DetermineState(loaded, time.Now(), false)
	assert.Equal(t, StateActive, state)
	assert.Equal(t, VolumeBandMedium, ParseVolumeBand(string(loaded.VolumeBand)))

	logLicensingFaultProof(t, "license_server_unreachable_last_known_good", map[string]string{
		"subsystem": "licensing",
		"state":     string(state),
	})
}

func signFaultJWT(t *testing.T, priv ed25519.PrivateKey, claims LicenseClaims) string {
	t.Helper()
	headerBytes, err := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "2026-01"})
	require.NoError(t, err)
	claimsBytes, err := json.Marshal(claims)
	require.NoError(t, err)
	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	sig := ed25519.Sign(priv, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}
