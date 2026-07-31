package domain

import (
	"errors"
	"strings"

	"espx/internal/config"
)

var (
	ErrInvalidRtbMode            = errors.New("rtb_mode must be off, shadow, or live")
	ErrInvalidRtbBudgetAuthority = errors.New("rtb_budget_authority must be rtb or lua")
)

const (
	SystemSettingRtbMode              = "rtb_mode"
	SystemSettingRtbBudgetAuthority   = "rtb_budget_authority"
)

func NormalizeRtbModeSetting(v string) (string, error) {
	switch config.ParseRtbMode(strings.TrimSpace(v)) {
	case config.RtbModeOff:
		return "off", nil
	case config.RtbModeShadow:
		return "shadow", nil
	case config.RtbModeLive:
		return "live", nil
	default:
		return "", ErrInvalidRtbMode
	}
}

func NormalizeRtbBudgetAuthoritySetting(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "rtb":
		return "rtb", nil
	case "lua", "redis":
		return "lua", nil
	case "":
		return "", nil
	default:
		return "", ErrInvalidRtbBudgetAuthority
	}
}
