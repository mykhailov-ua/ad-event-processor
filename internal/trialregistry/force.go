package trialregistry

import "strings"

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
