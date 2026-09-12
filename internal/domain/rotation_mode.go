package domain

import "strings"

type RotationMode string

const (
	RotationModeWeighted   RotationMode = "weighted"
	RotationModeUnseen     RotationMode = "unseen"
	RotationModeFixOn      RotationMode = "fix_on"
	RotationModeSequential RotationMode = "sequential"
)

func NormalizeRotationMode(raw string) RotationMode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "unseen":
		return RotationModeUnseen
	case "fix_on", "fix-on", "fixon":
		return RotationModeFixOn
	case "sequential", "top_to_bottom", "top-to-bottom":
		return RotationModeSequential
	default:
		return RotationModeWeighted
	}
}
