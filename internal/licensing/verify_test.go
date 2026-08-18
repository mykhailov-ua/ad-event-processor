package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestJWT(t *testing.T, privKey ed25519.PrivateKey, kid string, claims LicenseClaims) string {
	header := map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
		"kid": kid,
	}
	headerBytes, err := json.Marshal(header)
	assert.NoError(t, err)

	claimsBytes, err := json.Marshal(claims)
	assert.NoError(t, err)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsBytes)

	signingInput := headerB64 + "." + claimsB64
	sig := ed25519.Sign(privKey, []byte(signingInput))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64
}

func TestVerifyJWT(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      "lic-123",
		KeyID:        "2026-01",
		DeploymentID: "dep-123",
		CustomerName: "Test Buyer",
		Plan:         "growth",
		ValidFrom:    time.Now().Add(-1 * time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
	}
	claims.Limits.MaxRPS = 1000

	t.Run("Valid JWT", func(t *testing.T) {
		token := generateTestJWT(t, priv, "2026-01", claims)
		parsed, err := VerifyJWT(token, pub)
		assert.NoError(t, err)
		assert.Equal(t, "Test Buyer", parsed.CustomerName)
		assert.Equal(t, uint64(1000), parsed.Limits.MaxRPS)
	})

	t.Run("Tampered Signature", func(t *testing.T) {
		token := generateTestJWT(t, priv, "2026-01", claims)
		parts := strings.Split(token, ".")
		sigBytes, _ := base64.RawURLEncoding.DecodeString(parts[2])
		sigBytes[0] ^= 0xFF
		parts[2] = base64.RawURLEncoding.EncodeToString(sigBytes)
		tamperedToken := strings.Join(parts, ".")
		_, err := VerifyJWT(tamperedToken, pub)
		assert.Error(t, err)
	})

	t.Run("Tampered Payload", func(t *testing.T) {
		token := generateTestJWT(t, priv, "2026-01", claims)
		require.NotEmpty(t, token)

		otherClaims := claims
		otherClaims.Limits.MaxRPS = 999999
		otherBytes, err := json.Marshal(otherClaims)
		require.NoError(t, err)
		otherB64 := base64.RawURLEncoding.EncodeToString(otherBytes)

		dot := strings.Index(token, ".")
		lastDot := strings.LastIndex(token, ".")
		require.Greater(t, dot, 0)
		require.Greater(t, lastDot, dot)
		tamperedToken := token[:dot+1] + otherB64 + "." + token[lastDot+1:]

		_, err = VerifyJWT(tamperedToken, pub)
		assert.Error(t, err)
	})

	t.Run("Wrong Public Key", func(t *testing.T) {
		token := generateTestJWT(t, priv, "2026-01", claims)
		wrongPub, _, _ := ed25519.GenerateKey(rand.Reader)
		_, err := VerifyJWT(token, wrongPub)
		assert.Error(t, err)
	})
}
