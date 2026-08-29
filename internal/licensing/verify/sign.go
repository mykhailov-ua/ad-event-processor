package verify

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	entitlements "ad-event-processor/internal/licensing/entitlements"
)

const DefaultLicenseKeyID = "2026-01"

func SignJWT(claims entitlements.LicenseClaims, privKey ed25519.PrivateKey, kid string) (string, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return "", errors.New("invalid private key length")
	}
	if kid == "" {
		kid = DefaultLicenseKeyID
	}
	claims.KeyID = kid
	if claims.Issuer == "" {
		claims.Issuer = "ad-event-processor-license"
	}
	header := map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
		"kid": kid,
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := headerB64 + "." + claimsB64
	sig := ed25519.Sign(privKey, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func ParsePrivateKey(keyBytes []byte) (ed25519.PrivateKey, error) {
	raw := strings.TrimSpace(string(keyBytes))
	if raw == "" {
		return nil, errors.New("empty private key")
	}
	if decoded, err := hex.DecodeString(raw); err == nil {
		if len(decoded) == ed25519.SeedSize {
			return ed25519.NewKeyFromSeed(decoded), nil
		}
		if len(decoded) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(decoded), nil
		}
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		if len(decoded) == ed25519.SeedSize {
			return ed25519.NewKeyFromSeed(decoded), nil
		}
		if len(decoded) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(decoded), nil
		}
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(raw); err == nil {
		if len(decoded) == ed25519.SeedSize {
			return ed25519.NewKeyFromSeed(decoded), nil
		}
		if len(decoded) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(decoded), nil
		}
	}
	return nil, errors.New("invalid private key encoding")
}
