package licensing

import (
	"crypto/ed25519"
	"os"
	"strings"
	"time"
)

type VerifiedLicense struct {
	State        LicenseState
	Entitlements Entitlements
	Claims       *LicenseClaims
}

func EntitlementsFromClaims(claims *LicenseClaims) Entitlements {
	if claims == nil {
		return Entitlements{}
	}
	return Entitlements{
		VolumeBand: ParseVolumeBand(string(claims.VolumeBand)),
		Limits:     claims.Limits,
		Features:   claims.Features.Normalized(),
	}
}

func VerifyLicenseFile(path string, pubKey ed25519.PublicKey, hostFingerprint string, now time.Time) (VerifiedLicense, error) {
	var out VerifiedLicense
	out.State = StateExpired

	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return out, ErrInvalidTokenFormat
	}

	var claims *LicenseClaims
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

	state := DetermineState(claims, now, claims.Revoked)
	out.State = state
	out.Claims = claims
	out.Entitlements = EntitlementsFromClaims(claims)
	return out, nil
}
