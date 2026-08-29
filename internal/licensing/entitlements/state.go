package entitlements

import (
	"time"
)

func DetermineState(claims *LicenseClaims, now time.Time, revoked bool) LicenseState {
	if revoked || ClaimsRevoked(claims) {
		return StateRevoked
	}
	if claims == nil {
		return StateExpired
	}
	if now.Before(claims.ValidFrom) {
		return StateExpired
	}
	if now.Before(claims.ValidUntil) {
		return StateActive
	}
	graceDuration := time.Duration(claims.GraceDays) * 24 * time.Hour
	if now.Before(claims.ValidUntil.Add(graceDuration)) {
		return StateGrace
	}
	return StateExpired
}

func ClaimsRevoked(claims *LicenseClaims) bool {
	return claims != nil && claims.Revoked
}
