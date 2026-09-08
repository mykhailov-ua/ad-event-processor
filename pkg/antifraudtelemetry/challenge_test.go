package antifraudtelemetry

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChallengePoW_roundTrip_holdout(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	salt := []byte("0123456789abcdef")
	now := int64(1_700_000_000)
	token, err := MintChallengeWithSalt(secret, cid, salt, now, 30, 2)
	require.NoError(t, err)
	info, err := ParseChallengeToken(secret, token, cid, now+1)
	require.NoError(t, err)
	require.Equal(t, uint8(2), info.Difficulty)

	var nonce uint32
	found := false
	for nonce = 0; nonce < 500000; nonce++ {
		if VerifyPoW(info.Salt[:], nonce, info.Difficulty) {
			found = true
			break
		}
	}
	require.True(t, found)

	mac := ComputeTelemetryMACDerived(token, nonce, 3200, 180, 220, 0, 0)
	var hex [32]byte
	for i := 0; i < telemetryMACLen; i++ {
		hex[i*2] = "0123456789abcdef"[mac[i]>>4]
		hex[i*2+1] = "0123456789abcdef"[mac[i]&0xf]
	}
	require.True(t, VerifyTelemetryMACDerived(token, nonce, 3200, 180, 220, 0, 0, hex[:]))
}

func TestChallenge_expired_holdout(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	token, err := MintChallengeWithSalt(secret, cid, []byte("salt1234567890ab"), int64(100), 30, 2)
	require.NoError(t, err)
	_, err = ParseChallengeToken(secret, token, cid, 200)
	require.ErrorIs(t, err, ErrChallengeExpired)
}
