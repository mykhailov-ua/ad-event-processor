package licensing

import (
	"fmt"
	"time"
)

func TierUsageWarnings(limits Limits, activeCampaigns int, state LicenseState, validUntil, now time.Time, renewBeforeDays int) []string {
	var w []string
	if maxCampaigns := limits.MaxActiveCampaigns; maxCampaigns > 0 {
		n := uint64(activeCampaigns)
		if n >= maxCampaigns {
			w = append(w, fmt.Sprintf("Active campaign cap reached (%d/%d). Upgrade tier for more campaigns.", activeCampaigns, maxCampaigns))
		} else if maxCampaigns >= 5 && n*100/maxCampaigns >= 80 {
			w = append(w, fmt.Sprintf("Approaching active campaign cap (%d/%d).", activeCampaigns, maxCampaigns))
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
