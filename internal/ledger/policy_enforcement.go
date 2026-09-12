package ledger

import (
	"strings"
	"time"
)

const (
	EnforcementPauseCampaign      = "pause_campaign"
	EnforcementBlacklistPlacement = "blacklist_placement"
	EnforcementNotifyOnly         = "notify_only"
	EnforcementPlatformPause      = "platform_pause"

	defaultMarginGuardCooldownSec = 3600
	maxMarginGuardCooldownSec     = 604800
)

func NormalizeEnforcement(raw string) string {
	switch strings.TrimSpace(raw) {
	case EnforcementPauseCampaign, EnforcementBlacklistPlacement, EnforcementNotifyOnly, EnforcementPlatformPause:
		return strings.TrimSpace(raw)
	default:
		return ""
	}
}

func CampaignBreachEnforcement(policy *Policy) string {
	if policy == nil {
		return EnforcementPauseCampaign
	}
	if mode := NormalizeEnforcement(policy.Enforcement); mode != "" {
		switch mode {
		case EnforcementBlacklistPlacement:
			return EnforcementPauseCampaign
		case EnforcementPlatformPause:
			return EnforcementPauseCampaign
		default:
			return mode
		}
	}
	return EnforcementPauseCampaign
}

func PlacementBreachEnforcement(policy *Policy) string {
	if policy == nil {
		return EnforcementBlacklistPlacement
	}
	if mode := NormalizeEnforcement(policy.Enforcement); mode != "" {
		switch mode {
		case EnforcementPauseCampaign, EnforcementPlatformPause:
			return EnforcementPauseCampaign
		default:
			return mode
		}
	}
	return EnforcementBlacklistPlacement
}

func PolicyCooldownSec(policy *Policy) int {
	if policy == nil || policy.CooldownSec <= 0 {
		return defaultMarginGuardCooldownSec
	}
	if policy.CooldownSec > maxMarginGuardCooldownSec {
		return maxMarginGuardCooldownSec
	}
	return policy.CooldownSec
}

func PolicyWantsPlatformPause(policy *Policy) bool {
	if policy == nil {
		return false
	}
	mode := NormalizeEnforcement(policy.Enforcement)
	return policy.PlatformPause || mode == EnforcementPlatformPause
}

func WithinCooldown(lastPause time.Time, cooldownSec int, now time.Time) bool {
	if lastPause.IsZero() || cooldownSec <= 0 {
		return false
	}
	return now.Sub(lastPause) < time.Duration(cooldownSec)*time.Second
}
