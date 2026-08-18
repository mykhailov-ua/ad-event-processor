package trialregistry

import "strings"

// ValidateForceOverride checks --force flags against vendor env policy.
func ValidateForceOverride(force bool, reason string) error {
	if !force {
		return nil
	}
	if !ForceOverrideAllowed() {
		return ErrForceNotAllowed
	}
	if strings.TrimSpace(reason) == "" {
		return ErrForceReason
	}
	return nil
}
