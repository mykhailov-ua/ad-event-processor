package domain

import "strings"

type AttestationMode string

const (
	AttestationModeOff    AttestationMode = "off"
	AttestationModeLight  AttestationMode = "light"
	AttestationModeStrict AttestationMode = "strict"
)

func ParseAttestationMode(s string) AttestationMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "light":
		return AttestationModeLight
	case "strict":
		return AttestationModeStrict
	default:
		return AttestationModeOff
	}
}

func ResolveAttestationMode(mode AttestationMode, legacyEnabled bool) AttestationMode {
	if mode == AttestationModeLight || mode == AttestationModeStrict {
		return mode
	}
	if legacyEnabled {
		return AttestationModeStrict
	}
	return AttestationModeOff
}

func (m AttestationMode) RequiresProbe() bool {
	return m == AttestationModeLight || m == AttestationModeStrict
}
