package entitlements

import (
	"fmt"
	"time"
)

type TierUsageCounts struct {
	Tenants uint64
	APIKeys uint64
}

func TierUsageWarnings(limits Limits, counts TierUsageCounts, activeCampaigns int, state LicenseState, validUntil, now time.Time, renewBeforeDays int) []string {
	var w []string
	if maxCampaigns := limits.MaxActiveCampaigns; maxCampaigns > 0 {
		n := uint64(activeCampaigns)
		if n >= maxCampaigns {
			w = append(w, fmt.Sprintf("Active campaign cap reached (%d/%d). Upgrade tier for more campaigns.", activeCampaigns, maxCampaigns))
		} else if maxCampaigns >= 5 && n*100/maxCampaigns >= 80 {
			w = append(w, fmt.Sprintf("Approaching active campaign cap (%d/%d).", activeCampaigns, maxCampaigns))
		}
	}
	if maxTenants := limits.MaxTenants; maxTenants > 0 && maxTenants < 999999 {
		if counts.Tenants >= maxTenants {
			w = append(w, fmt.Sprintf("Client workspace cap reached (%d/%d). Upgrade tier for more workspaces.", counts.Tenants, maxTenants))
		} else if maxTenants >= 3 && counts.Tenants*100/maxTenants >= 80 {
			w = append(w, fmt.Sprintf("Approaching client workspace cap (%d/%d).", counts.Tenants, maxTenants))
		}
	}
	if maxKeys := limits.MaxAPIKeys; maxKeys > 0 && maxKeys < 999999 {
		if counts.APIKeys >= maxKeys {
			w = append(w, fmt.Sprintf("API key cap reached (%d/%d). Upgrade tier for more keys.", counts.APIKeys, maxKeys))
		} else if maxKeys >= 3 && counts.APIKeys*100/maxKeys >= 80 {
			w = append(w, fmt.Sprintf("Approaching API key cap (%d/%d).", counts.APIKeys, maxKeys))
		}
	}
	switch state {
	case StateGrace:
		w = append(w, "License grace period - paste renewal JWT in Settings.")
	case StateOfflineWarn:
		w = append(w, "License heartbeat offline - reconnect or renew soon.")
	case StateOfflineGrace:
		w = append(w, "License offline grace ending - renew JWT to avoid ingest block.")
	}
	if state == StateActive && !validUntil.IsZero() && renewBeforeDays > 0 {
		days := int(validUntil.Sub(now).Hours() / 24)
		if days >= 0 && days <= renewBeforeDays {
			w = append(w, fmt.Sprintf("License renews in %d day(s) - request USDT invoice early.", days))
		}
	}
	return w
}
