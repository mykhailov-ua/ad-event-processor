package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrClickFilterTierRedirectOnlyUnlicensed = errors.New("redirect_only click filter tier is not licensed on this deployment")

type ClickFilterTier string

const (
	ClickFilterTierFull         ClickFilterTier = "full"
	ClickFilterTierLight        ClickFilterTier = "light"
	ClickFilterTierRedirectOnly ClickFilterTier = "redirect_only"
)

func NormalizeClickFilterTier(raw string) ClickFilterTier {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "full":
		return ClickFilterTierFull
	case "light":
		return ClickFilterTierLight
	case "redirect_only", "redirect-only":
		return ClickFilterTierRedirectOnly
	default:
		return ClickFilterTierFull
	}
}

func ClickFilterTierSkipsStreamPublish(tier ClickFilterTier) bool {
	return tier == ClickFilterTierLight || tier == ClickFilterTierRedirectOnly
}

func CampaignRequiresFullClickFilters(c *Campaign) bool {
	if c == nil {
		return false
	}
	if c.SilentRejectEnabled {
		return true
	}
	if c.TLSFingerprintBlockEnabled {
		return true
	}
	if c.ProxyVPNBlockEnabled {
		return true
	}
	if c.CIDRBlockEnabled {
		return true
	}
	if c.ModeratorIntelEnabled {
		return true
	}
	if c.BehaviorFlags != 0 {
		return true
	}
	if c.RetargetSegmentID != uuid.Nil {
		return true
	}
	if c.SegmentIncludeID != uuid.Nil || c.SegmentExcludeID != uuid.Nil {
		return true
	}
	if c.RequireConsentPurposes != 0 {
		return true
	}
	return false
}

func ResolveClickFilterTier(c *Campaign, envDefault string, redirectOnlyLicensed bool) ClickFilterTier {
	tier := ClickFilterTierFull
	if c != nil && strings.TrimSpace(c.ClickFilterTier) != "" {
		tier = NormalizeClickFilterTier(c.ClickFilterTier)
	} else {
		tier = NormalizeClickFilterTier(envDefault)
	}
	if tier == ClickFilterTierRedirectOnly && !redirectOnlyLicensed {
		return ClickFilterTierFull
	}
	if tier != ClickFilterTierFull && CampaignRequiresFullClickFilters(c) {
		return ClickFilterTierFull
	}
	return tier
}

func ValidateClickFilterTierForSave(raw string, redirectOnlyLicensed bool) error {
	tier := NormalizeClickFilterTier(raw)
	if tier == ClickFilterTierRedirectOnly && !redirectOnlyLicensed {
		return ErrClickFilterTierRedirectOnlyUnlicensed
	}
	return nil
}
