package config

import "strings"

type RtbMode string

const (
	RtbModeOff    RtbMode = "off"
	RtbModeShadow RtbMode = "shadow"
	RtbModeLive   RtbMode = "live"
)

func ParseRtbMode(raw string) RtbMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "shadow":
		return RtbModeShadow
	case "live":
		return RtbModeLive
	default:
		return RtbModeOff
	}
}

func (c *Config) RtbEnabled() bool {
	return c != nil && ParseRtbMode(c.RtbMode) != RtbModeOff
}

func (c *Config) RtbLiveSelectsCampaign() bool {
	return c != nil && ParseRtbMode(c.RtbMode) == RtbModeLive
}

func (c *Config) RtbBudgetAuthoritative() bool {
	if c == nil || !c.RtbLiveSelectsCampaign() {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(c.RtbBudgetAuthority), "rtb")
}

func (c *Config) RtbTargetingIndexEnabled() bool {
	return c != nil && c.RtbTargetingIndex
}

func (c *Config) RtbPrebidIVTEnabled() bool {
	return c != nil && c.RtbPrebidIVT
}
