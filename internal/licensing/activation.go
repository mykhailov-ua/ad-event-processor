package licensing

import (
	"errors"
	"time"

	"ad-event-processor/pkg/coldpath"
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
	capped := now.UTC().Add(HeartbeatJWTMaxTTL)
	if licenseValidUntil.Before(capped) {
		return licenseValidUntil.UTC()
	}
	return capped
}

func NormalizeMaxActivations(maxActivations int32) int {
	if maxActivations <= 0 {
		return 1
	}
	return int(maxActivations)
}

func EvaluateActivate(fingerprint, licenseKey string, maxActivations int32, activations []ActivationRecord, deployment *DeploymentRecord) ActivationDecision {
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

func EvaluateHeartbeat(fingerprint, licenseKey string, activations []ActivationRecord, deployment *DeploymentRecord) ActivationDecision {
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
	fps := make([]string, 0, len(activations)+1)
	for _, act := range activations {
		fps = append(fps, act.Fingerprint)
	}
	if extra != "" {
		fps = append(fps, extra)
	}
	return len(coldpath.UniqueSlice(fps))
}
