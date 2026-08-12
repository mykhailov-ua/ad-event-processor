package licensing

import (
	"fmt"
	"time"
)

// TierUsageWarnings returns admin UI banner hints for tier caps and renewal windows.
func TierUsageWarnings(limits Limits, activeCampaigns int, state LicenseState, validUntil, now time.Time, renewBeforeDays int) []string {
	var w []string
	if max := limits.MaxActiveCampaigns; max > 0 {
		n := uint64(activeCampaigns)
		if n >= max {
			w = append(w, fmt.Sprintf("Active campaign cap reached (%d/%d). Upgrade tier for more campaigns.", activeCampaigns, max))
		} else if max >= 5 && n*100/max >= 80 {
			w = append(w, fmt.Sprintf("Approaching active campaign cap (%d/%d).", activeCampaigns, max))
		}
	}
	switch state {
	case StateGrace:
		w = append(w, "License grace period — paste renewal JWT in Settings.")
	case StateOfflineWarn:
		w = append(w, "License heartbeat offline — reconnect or renew soon.")
	case StateOfflineGrace:
		w = append(w, "License offline grace ending — renew JWT to avoid ingest block.")
	}
	if state == StateActive && !validUntil.IsZero() && renewBeforeDays > 0 {
		days := int(validUntil.Sub(now).Hours() / 24)
		if days >= 0 && days <= renewBeforeDays {
			w = append(w, fmt.Sprintf("License renews in %d day(s) — request USDT invoice early.", days))
		}
	}
	return w
}
