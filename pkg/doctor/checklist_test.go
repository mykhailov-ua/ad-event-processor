package doctor

import (
	"os"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
)

func TestMVSSChecklistTelemetryDefault(t *testing.T) {
	t.Setenv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"), "")
	_ = os.Unsetenv(naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN"))

	rows := MVSSChecklist(nil)
	var telemetry ChecklistRow
	for _, row := range rows {
		if row.ID == "telemetry_opt_in" {
			telemetry = row
			break
		}
	}
	if telemetry.Status != StatusPass {
		t.Fatalf("telemetry status=%s want pass", telemetry.Status)
	}
}

func TestCheckPIISaltPlaceholder(t *testing.T) {
	t.Setenv("PII_SALT_HEX", "deadbeef")
	row := checkPIISalt()
	if row.Status != StatusFail {
		t.Fatalf("status=%s want fail", row.Status)
	}
}

func TestChecklistExitCodeWarn(t *testing.T) {
	code := ChecklistExitCode([]ChecklistRow{{ID: "x", Status: StatusWarn}})
	if code != 1 {
		t.Fatalf("code=%d want 1", code)
	}
}
