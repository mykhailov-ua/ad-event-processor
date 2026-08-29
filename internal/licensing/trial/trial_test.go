package trial_test

import (
	"testing"

	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensing/trial"
)

func TestUpgradePlanForLicense_pilotAndUnconfigured(t *testing.T) {
	t.Parallel()
	if got := trial.UpgradePlanForLicense("pilot", "ACTIVE"); got != entitlements.SKUCodeStarter {
		t.Fatalf("pilot: got %q want starter", got)
	}
	if got := trial.UpgradePlanForLicense("", "UNCONFIGURED"); got != entitlements.SKUCodeStarter {
		t.Fatalf("unconfigured: got %q want starter", got)
	}
	if got := trial.UpgradePlanForLicense("starter", "ACTIVE"); got != "" {
		t.Fatalf("starter: got %q want empty", got)
	}
}
