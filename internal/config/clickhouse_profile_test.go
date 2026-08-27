package config

import (
	"os"
	"testing"
)

func TestClickHouseEnabled(t *testing.T) {
	cfg := &Config{ClickHouseEnabled: true, ClickHouseDSN: "clickhouse://127.0.0.1/db"}
	if !cfg.IsClickHouseEnabled() {
		t.Fatal("expected enabled with DSN and ClickHouseEnabled")
	}

	cfg.ClickHouseEnabled = false
	if cfg.IsClickHouseEnabled() {
		t.Fatal("expected disabled when ClickHouseEnabled=false")
	}

	cfg.ClickHouseEnabled = true
	cfg.ClickHouseDSN = ""
	if cfg.IsClickHouseEnabled() {
		t.Fatal("expected disabled without DSN")
	}
}

func TestClickHouseEnabledFromEnv(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want bool
	}{
		{"", true},
		{"1", true},
		{"0", false},
		{"false", false},
		{"off", false},
	} {
		t.Run(tc.env, func(t *testing.T) {
			if tc.env == "" {
				_ = os.Unsetenv("CH_ENABLED")
			} else {
				t.Setenv("CH_ENABLED", tc.env)
			}
			if got := clickHouseEnabledFromEnv(); got != tc.want {
				t.Fatalf("clickHouseEnabledFromEnv()=%v want %v", got, tc.want)
			}
		})
	}
}
