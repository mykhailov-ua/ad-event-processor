package ingestion

import (
	"strings"

	"espx/internal/config"

	"espx/internal/domain"
)

var ErrInvalidRtbBudgetAuthority = domain.ErrInvalidRtbBudgetAuthority

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
	return domain.NormalizeRtbBudgetAuthoritySetting(v)
}
