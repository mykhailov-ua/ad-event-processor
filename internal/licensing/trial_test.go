package licensing

import "testing"

func TestUpgradePlanForLicense_pilotAndUnconfigured(t *testing.T) {
	t.Parallel()
	if got := UpgradePlanForLicense("pilot", "ACTIVE"); got != SKUCodeStarter {
		t.Fatalf("pilot: got %q want starter", got)
	}
	if got := UpgradePlanForLicense("", "UNCONFIGURED"); got != SKUCodeStarter {
		t.Fatalf("unconfigured: got %q want starter", got)
	}
	if got := UpgradePlanForLicense("starter", "ACTIVE"); got != "" {
		t.Fatalf("starter: got %q want empty", got)
	}
}
