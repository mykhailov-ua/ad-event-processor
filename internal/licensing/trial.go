package licensing

import "strings"

const PilotTrialValidDays = 14

func UpgradePlanForLicense(planCode, state string) string {
	plan := strings.ToLower(strings.TrimSpace(planCode))
	st := strings.ToUpper(strings.TrimSpace(state))
	if st == "UNCONFIGURED" || plan == "" || plan == SKUCodePilot {
		return SKUCodeStarter
	}
	return ""
}
