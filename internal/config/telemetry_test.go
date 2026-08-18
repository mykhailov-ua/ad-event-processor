package config

import (
	"os"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
)

func TestTelemetryOptInFromEnv(t *testing.T) {
	_ = os.Unsetenv("ADSTACK_TELEMETRY_OPT_IN")
	_ = os.Unsetenv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"))
	if telemetryOptInFromEnv() {
		t.Fatal("unset must default off")
	}
	t.Setenv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"), "0")
	if telemetryOptInFromEnv() {
		t.Fatal("0 must be off")
	}
	t.Setenv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"), "1")
	if !telemetryOptInFromEnv() {
		t.Fatal("1 must enable opt-in")
	}
	_ = os.Unsetenv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"))
	t.Setenv("ADSTACK_TELEMETRY_OPT_IN", "1")
	if !telemetryOptInFromEnv() {
		t.Fatal("ADSTACK_TELEMETRY_OPT_IN must enable opt-in")
	}
}
