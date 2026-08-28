package doctor

import (
	"os"
	"strings"
	"testing"
)

func TestDeployProfileChecklistUnknown(t *testing.T) {
	rows := DeployProfileChecklist("invalid", nil)
	if len(rows) != 1 || rows[0].Status != StatusFail {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestDeployProfileChecklistIngestOnlyEnv(t *testing.T) {
	t.Setenv("CH_ENABLED", "0")
	t.Setenv("CONTROL_ENABLE_PAYMENT", "0")
	t.Setenv("CONTROL_ENABLE_BILLING", "0")
	t.Setenv("CONTROL_ENABLE_NOTIFIER", "0")
	t.Setenv("DB_DSN", "postgres://u:p@localhost/db?sslmode=disable")
	t.Setenv("REDIS_PASSWORD", "secret")

	rows := DeployProfileChecklist(ProfileIngestOnly, nil)
	for _, row := range rows {
		if strings.HasPrefix(row.ID, "running_") {
			continue
		}
		if row.Status == StatusFail && row.ID != "compose_include_control" {
			t.Fatalf("unexpected fail %s: %s", row.ID, row.Detail)
		}
	}
}

func TestDeployProfileChecklistIngestOnlyRejectsCH(t *testing.T) {
	t.Setenv("CH_ENABLED", "1")
	t.Setenv("CONTROL_ENABLE_PAYMENT", "0")
	t.Setenv("CONTROL_ENABLE_BILLING", "0")
	t.Setenv("CONTROL_ENABLE_NOTIFIER", "0")
	t.Setenv("DB_DSN", "postgres://u:p@localhost/db?sslmode=disable")
	t.Setenv("REDIS_PASSWORD", "secret")

	rows := DeployProfileChecklist(ProfileIngestOnly, nil)
	var ch Status
	for _, row := range rows {
		if row.ID == "ch_enabled" {
			ch = row.Status
		}
	}
	if ch != StatusFail {
		t.Fatalf("ch_enabled status=%s want fail", ch)
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("TEST_BOOL", "0")
	got, unset := envBool("TEST_BOOL")
	if unset || got {
		t.Fatalf("got=%v unset=%v", got, unset)
	}
	_ = os.Unsetenv("TEST_BOOL")
	_, unset = envBool("TEST_BOOL_MISSING")
	if !unset {
		t.Fatal("expected unset")
	}
}
