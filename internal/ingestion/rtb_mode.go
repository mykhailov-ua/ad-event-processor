package ingestion

import (
	"strings"

	"ad-event-processor/internal/config"

	"ad-event-processor/internal/domain"
)

const SystemSettingRtbMode = domain.SystemSettingRtbMode

var ErrInvalidRtbMode = domain.ErrInvalidRtbMode

func NormalizeRtbModeSetting(v string) (string, error) {
	return domain.NormalizeRtbModeSetting(v)
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
