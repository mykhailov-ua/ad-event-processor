package ingestion

import (
	"errors"
	"strings"

	"espx/internal/config"
)

var ErrInvalidRtbMode = errors.New("rtb_mode must be off, shadow, or live")

const SystemSettingRtbMode = "rtb_mode"

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

func RtbModeFromSetting(setting string, cfg *config.Config) uint8 {
	raw := strings.TrimSpace(setting)
	if raw == "" && cfg != nil {
		return rtbModeFromConfig(cfg)
	}
	switch config.ParseRtbMode(raw) {
	case config.RtbModeShadow:
		return rtbModeShadow
	case config.RtbModeLive:
		return rtbModeLive
	default:
		return rtbModeOff
	}
}
