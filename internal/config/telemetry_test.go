package config

import (
	"os"
	"testing"
)

func TestTelemetryOptInFromEnv(t *testing.T) {
	_ = os.Unsetenv("ESPX_TELEMETRY_OPT_IN")
	if telemetryOptInFromEnv() {
		t.Fatal("unset must default off")
	}
	t.Setenv("ESPX_TELEMETRY_OPT_IN", "0")
	if telemetryOptInFromEnv() {
		t.Fatal("0 must be off")
	}
	t.Setenv("ESPX_TELEMETRY_OPT_IN", "1")
	if !telemetryOptInFromEnv() {
		t.Fatal("1 must enable opt-in")
	}
}
