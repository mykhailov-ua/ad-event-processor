package licensing

import "time"

type LicenseDiagnostics struct {
	State            LicenseState
	DeploymentID     string
	ValidUntil       time.Time
	DaysToExpiry     int
	HostFingerprint  string
	BindFingerprint  string
	BindMode         string
	FingerprintMatch bool
	Revoked          bool
}

func BuildLicenseDiagnostics(claims *LicenseClaims, state LicenseState, now time.Time) LicenseDiagnostics {
	var d LicenseDiagnostics
	d.State = state
	d.HostFingerprint = HostFingerprint()
	if claims != nil {
		d.DeploymentID = claims.DeploymentID
		d.ValidUntil = claims.ValidUntil
		d.BindFingerprint = claims.Bind.Fingerprint
		d.BindMode = claims.Bind.Mode
		d.Revoked = claims.Revoked
		if !claims.ValidUntil.IsZero() {
			days := int(claims.ValidUntil.Sub(now).Hours() / 24)
			if days < 0 {
				days = 0
			}
			d.DaysToExpiry = days
		}
		if BindModeHard(claims.Bind.Mode) && claims.Bind.Fingerprint != "" {
			d.FingerprintMatch = d.HostFingerprint == claims.Bind.Fingerprint
		} else {
			d.FingerprintMatch = true
		}
	}
	return d
}

func ClaimsRevoked(claims *LicenseClaims) bool {
	return claims != nil && claims.Revoked
}
