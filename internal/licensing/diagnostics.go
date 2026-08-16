package licensing

import "time"

type LicenseDiagnostics struct {
	State            LicenseState
	DeploymentID     string
	ValidUntil       time.Time
	DaysToExpiry     int
	HostFingerprint  string
	HostHWID         string
	BindFingerprint  string
	BindHWIDHash     string
	BindMode         string
	FingerprintMatch bool
	HWIDMatch        bool
	Revoked          bool
}

func BuildLicenseDiagnostics(claims *LicenseClaims, state LicenseState, now time.Time) LicenseDiagnostics {
	var d LicenseDiagnostics
	d.State = state
	d.HostFingerprint = HostFingerprint()
	d.HostHWID = HostHWID()
	if claims != nil {
		d.DeploymentID = claims.DeploymentID
		d.ValidUntil = claims.ValidUntil
		d.BindFingerprint = claims.Bind.Fingerprint
		d.BindHWIDHash = claims.HWIDHash
		d.BindMode = claims.Bind.Mode
		d.Revoked = claims.Revoked
		if !claims.ValidUntil.IsZero() {
			days := int(claims.ValidUntil.Sub(now).Hours() / 24)
			if days < 0 {
				days = 0
			}
			d.DaysToExpiry = days
		}
		if BindModeHard(claims.Bind.Mode) {
			switch {
			case claims.HWIDHash != "":
				d.HWIDMatch = d.HostHWID == claims.HWIDHash
				d.FingerprintMatch = d.HWIDMatch
			case claims.Bind.Fingerprint != "":
				d.FingerprintMatch = d.HostFingerprint == claims.Bind.Fingerprint
				d.HWIDMatch = d.FingerprintMatch
			default:
				d.FingerprintMatch = true
				d.HWIDMatch = true
			}
		} else {
			d.FingerprintMatch = true
			d.HWIDMatch = true
		}
	}
	return d
}

func ClaimsRevoked(claims *LicenseClaims) bool {
	return claims != nil && claims.Revoked
}
