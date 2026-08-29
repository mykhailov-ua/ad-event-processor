package trial

import (
	"strings"

	"ad-event-processor/internal/licensing/entitlements"
)

const PilotTrialValidDays = 14

func UpgradePlanForLicense(planCode, state string) string {
	plan := strings.ToLower(strings.TrimSpace(planCode))
	st := strings.ToUpper(strings.TrimSpace(state))
	if st == "UNCONFIGURED" || plan == "" || plan == entitlements.SKUCodePilot {
		return entitlements.SKUCodeStarter
	}
	return ""
}
