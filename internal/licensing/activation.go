package licensing

import (
	"errors"
	"time"
)

const HeartbeatJWTMaxTTL = 72 * time.Hour

var (
	ErrFingerprintRequired = errors.New("fingerprint required")
	ErrFingerprintMismatch = errors.New("fingerprint mismatch")
	ErrActivationLimit     = errors.New("activation limit exceeded")
	ErrDeploymentMismatch  = errors.New("deployment license mismatch")
	ErrDeploymentNotFound  = errors.New("deployment not found")
)

type ActivationRecord struct {
	LicenseKey   string
	Fingerprint  string
	DeploymentID string
}

type DeploymentRecord struct {
	DeploymentID string
	LicenseKey   string
	Fingerprint  string
}

type ActivationDecision struct {
	Allow                bool
	DenyReason           string
	BindActivation       bool
	FlagManyFingerprints bool
}

func CapHeartbeatValidUntil(licenseValidUntil, now time.Time) time.Time {
	cap := now.UTC().Add(HeartbeatJWTMaxTTL)
	if licenseValidUntil.Before(cap) {
		return licenseValidUntil.UTC()
	}
	return cap
}

func NormalizeMaxActivations(max int32) int {
	if max <= 0 {
		return 1
	}
	return int(max)
}

func EvaluateActivate(fingerprint string, licenseKey string, maxActivations int32, activations []ActivationRecord, deployment *DeploymentRecord) ActivationDecision {
	if fingerprint == "" {
		return ActivationDecision{Allow: false, DenyReason: ErrFingerprintRequired.Error()}
	}
	limit := NormalizeMaxActivations(maxActivations)

	if deployment != nil {
		if deployment.LicenseKey != licenseKey {
			return ActivationDecision{Allow: false, DenyReason: ErrDeploymentMismatch.Error()}
		}
		if deployment.Fingerprint != "" && deployment.Fingerprint != fingerprint {
			return ActivationDecision{Allow: false, DenyReason: ErrFingerprintMismatch.Error()}
		}
	}

	matched := false
	for _, act := range activations {
		if act.Fingerprint == fingerprint {
			matched = true
			break
		}
	}
	if !matched && len(activations) >= limit {
		return ActivationDecision{Allow: false, DenyReason: ErrActivationLimit.Error()}
	}

	flagMany := distinctFingerprints(activations, fingerprint) > 1
	return ActivationDecision{
		Allow:                true,
		BindActivation:       !matched,
		FlagManyFingerprints: flagMany,
	}
}

func EvaluateHeartbeat(fingerprint string, licenseKey string, activations []ActivationRecord, deployment *DeploymentRecord) ActivationDecision {
	if fingerprint == "" {
		return ActivationDecision{Allow: false, DenyReason: ErrFingerprintRequired.Error()}
	}
	if deployment == nil {
		return ActivationDecision{Allow: false, DenyReason: ErrDeploymentNotFound.Error()}
	}
	if deployment.LicenseKey != licenseKey {
		return ActivationDecision{Allow: false, DenyReason: ErrDeploymentMismatch.Error()}
	}
	if deployment.Fingerprint != "" && deployment.Fingerprint != fingerprint {
		return ActivationDecision{Allow: false, DenyReason: ErrFingerprintMismatch.Error()}
	}

	matched := false
	for _, act := range activations {
		if act.Fingerprint == fingerprint {
			matched = true
			break
		}
	}
	if !matched && len(activations) > 0 {
		return ActivationDecision{Allow: false, DenyReason: ErrFingerprintMismatch.Error()}
	}

	flagMany := distinctFingerprints(activations, fingerprint) > 1
	return ActivationDecision{
		Allow:                true,
		BindActivation:       !matched,
		FlagManyFingerprints: flagMany,
	}
}

func distinctFingerprints(activations []ActivationRecord, extra string) int {
	seen := make(map[string]struct{}, len(activations)+1)
	for _, act := range activations {
		if act.Fingerprint == "" {
			continue
		}
		seen[act.Fingerprint] = struct{}{}
	}
	if extra != "" {
		seen[extra] = struct{}{}
	}
	return len(seen)
}
