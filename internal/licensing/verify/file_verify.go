package verify

import (
	"crypto/ed25519"
	"os"
	"strings"
	"time"

	entitlements "ad-event-processor/internal/licensing/entitlements"
)

type VerifiedLicense struct {
	State        entitlements.LicenseState
	Entitlements entitlements.Entitlements
	Claims       *entitlements.LicenseClaims
}

func EntitlementsFromClaims(claims *entitlements.LicenseClaims) entitlements.Entitlements {
	if claims == nil {
		return entitlements.Entitlements{}
	}
	features := entitlements.SanitizeFeaturesForSKU(claims.SKU, claims.Features)
	return entitlements.Entitlements{
		VolumeBand: entitlements.VolumeBand(claims.VolumeBand),
		Limits:     claims.Limits,
		Features:   features.Normalized(),
	}
}

func VerifyLicenseFile(path string, pubKey ed25519.PublicKey, hostFingerprint string, now time.Time) (VerifiedLicense, error) {
	var out VerifiedLicense
	out.State = entitlements.StateExpired

	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return out, ErrInvalidTokenFormat
	}

	var claims *entitlements.LicenseClaims
	if len(pubKey) > 0 {
		claims, err = VerifyJWT(token, pubKey)
	} else {
		claims, err = VerifyJWTResolved(token)
	}
	if err != nil {
		return out, err
	}
	if err := VerifyDeploymentBind(claims, hostFingerprint); err != nil {
		return out, err
	}

	state := entitlements.DetermineState(claims, now, claims.Revoked)
	out.State = state
	out.Claims = claims
	out.Entitlements = EntitlementsFromClaims(claims)
	return out, nil
}
