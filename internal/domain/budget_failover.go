package domain

import (
	"net/url"
	"strings"
)

type BudgetFailoverMode string

const (
	BudgetFailoverModeNone        BudgetFailoverMode = "none"
	BudgetFailoverModeFallbackURL BudgetFailoverMode = "fallback_url"
	BudgetFailoverModeFlowNext    BudgetFailoverMode = "flow_next"
)

type ClickFilterBudgetPolicy string

const (
	ClickFilterBudgetPolicyInherit           ClickFilterBudgetPolicy = "inherit"
	ClickFilterBudgetPolicyFull              ClickFilterBudgetPolicy = "full"
	ClickFilterBudgetPolicyLightSkipDebit    ClickFilterBudgetPolicy = "light_skip_debit"
	ClickFilterBudgetPolicyRedirectSkipDebit ClickFilterBudgetPolicy = "redirect_skip_debit"
)

func NormalizeBudgetFailoverMode(raw string) BudgetFailoverMode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "fallback_url", "fallback-url":
		return BudgetFailoverModeFallbackURL
	case "flow_next", "flow-next":
		return BudgetFailoverModeFlowNext
	default:
		return BudgetFailoverModeNone
	}
}

func NormalizeClickFilterBudgetPolicy(raw string) ClickFilterBudgetPolicy {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "inherit":
		return ClickFilterBudgetPolicyInherit
	case "full":
		return ClickFilterBudgetPolicyFull
	case "light_skip_debit", "light-skip-debit":
		return ClickFilterBudgetPolicyLightSkipDebit
	case "redirect_skip_debit", "redirect-skip-debit":
		return ClickFilterBudgetPolicyRedirectSkipDebit
	default:
		return ClickFilterBudgetPolicyInherit
	}
}

func ApplyClickFilterBudgetPolicy(tier ClickFilterTier, policy ClickFilterBudgetPolicy) ClickFilterTier {
	switch policy {
	case ClickFilterBudgetPolicyFull:
		return ClickFilterTierFull
	case ClickFilterBudgetPolicyLightSkipDebit:
		return ClickFilterTierLight
	case ClickFilterBudgetPolicyRedirectSkipDebit:
		return ClickFilterTierRedirectOnly
	default:
		return tier
	}
}

func ValidHTTPSRedirectURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return false
	}
	return true
}
