package ingestion

import (
	"errors"
	"strings"

	"espx/internal/config"
)

var ErrInvalidRtbBudgetAuthority = errors.New("rtb_budget_authority must be rtb or lua")

const systemSettingRtbBudgetAuthority = "rtb_budget_authority"

func BudgetAuthorityFromSettings(cfg *config.Config, setting string) BudgetAuthority {
	if cfg == nil || !cfg.RtbEnabled() {
		return BudgetAuthorityShadow
	}
	if !cfg.RtbLiveSelectsCampaign() {
		return BudgetAuthorityShadow
	}
	raw := strings.TrimSpace(setting)
	if raw == "" {
		raw = cfg.RtbBudgetAuthority
	}
	switch strings.ToLower(raw) {
	case "rtb":
		return BudgetAuthorityRTB
	case "lua", "redis", "":
		return BudgetAuthorityRedis
	default:
		return BudgetAuthorityRedis
	}
}

func RtbSkipLuaBudgetDebit(cfg *config.Config, setting string) bool {
	return BudgetAuthorityFromSettings(cfg, setting) == BudgetAuthorityRTB
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
