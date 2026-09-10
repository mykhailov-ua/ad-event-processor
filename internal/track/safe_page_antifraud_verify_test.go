package track

import (
	"encoding/json"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/antifraudtelemetry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEvaluateSafePageAntifraudCrypto_roundTrip_holdout(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	now := int64(1_700_000_000)
	token, err := antifraudtelemetry.MintChallengeWithSalt(secret, cid, []byte("0123456789abcdef"), now, 30, 2)
	require.NoError(t, err)

	var nonce uint32
	found := false
	info, err := antifraudtelemetry.ParseChallengeToken(secret, token, cid, now+1)
	require.NoError(t, err)
	for nonce = 0; nonce < 500000; nonce++ {
		if antifraudtelemetry.VerifyPoW(info.Salt[:], nonce, info.Difficulty) {
			found = true
			break
		}
	}
	require.True(t, found)

	dwellMs := uint32(3200)
	pointerCV := uint16(180)
	rafCV := uint16(220)
	mac := antifraudtelemetry.ComputeTelemetryMACDerived(token, nonce, dwellMs, pointerCV, rafCV, 0, 0)
	var macHex [32]byte
	for i := 0; i < 16; i++ {
		macHex[i*2] = "0123456789abcdef"[mac[i]>>4]
		macHex[i*2+1] = "0123456789abcdef"[mac[i]&0xf]
	}

	raw, err := json.Marshal(map[string]interface{}{
		"challenge_token":  token,
		"pow_nonce":        nonce,
		"telemetry_mac":    string(macHex[:]),
		"dwell_ms":         dwellMs,
		"pointer_cv_milli": pointerCV,
		"raf_cv_milli":     rafCV,
	})
	require.NoError(t, err)

	snap, ok := ParseAntifraudSnapshotFromJSON(raw)
	require.True(t, ok)
	fail, code := EvaluateSafePageAntifraudCrypto(cid, snap, secret, now+1)
	require.False(t, fail)
	require.Equal(t, "", code)
}

func TestEvaluateSafePageAntifraudCrypto_badMAC_holdout(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	now := int64(1_700_000_000)
	token, err := antifraudtelemetry.MintChallengeWithSalt(secret, cid, []byte("salt1234567890ab"), now, 30, 2)
	require.NoError(t, err)

	raw, err := json.Marshal(map[string]interface{}{
		"challenge_token":  token,
		"pow_nonce":        uint32(1),
		"telemetry_mac":    "00000000000000000000000000000000",
		"dwell_ms":         uint32(1000),
		"pointer_cv_milli": uint16(100),
		"raf_cv_milli":     uint16(100),
	})
	require.NoError(t, err)

	snap, ok := ParseAntifraudSnapshotFromJSON(raw)
	require.True(t, ok)
	fail, code := EvaluateSafePageAntifraudCrypto(cid, snap, secret, now+1)
	require.True(t, fail)
	require.Equal(t, safePageAttestAntifraudPowInvalid, code)
}

func TestRequiresSafePageAntifraudCrypto_strictBundle_holdout(t *testing.T) {
	camp := &domain.Campaign{
		SafePageEnabled:    true,
		AttestationEnabled: true,
		AttestationMode:    domain.AttestationModeStrict,
	}
	require.False(t, RequiresSafePageAntifraudCrypto(camp, true))
}

func TestSafePageHydrator_holdout_mergesBiometricsEvents(t *testing.T) {
	src := string(safePageHydratorJS)
	require.Contains(t, src, "trackBiometricsSnapshot")
	require.Contains(t, src, "trackBiometricsArm")
}
