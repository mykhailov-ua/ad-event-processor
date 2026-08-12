package licensing

import (
	"strings"
)

func BindModeHard(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "hard", "fingerprint":
		return true
	default:
		return false
	}
}

func BindModeMulti(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), "multi")
}

func VerifyDeploymentBind(claims *LicenseClaims, hostFingerprint string) error {
	if claims == nil {
		return ErrInvalidTokenFormat
	}
	if !BindModeHard(claims.Bind.Mode) {
		return nil
	}
	expected := strings.TrimSpace(claims.Bind.Fingerprint)
	if expected == "" {
		return nil
	}
	if hostFingerprint == "" {
		return ErrFingerprintRequired
	}
	if hostFingerprint != expected {
		return ErrFingerprintMismatch
	}
	return nil
}
