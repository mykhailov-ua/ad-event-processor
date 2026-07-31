package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
)

var legacyEnvWarned sync.Map

func warnLegacyEnvOnce(legacyKey, adstackKey string) {
	if _, loaded := legacyEnvWarned.LoadOrStore(legacyKey, true); loaded {
		return
	}
	slog.Warn("deprecated environment variable", "legacy", legacyKey, "use", adstackKey)
}

func envStringDual(adstackKey, legacyKey string) string {
	if v, ok := os.LookupEnv(adstackKey); ok && strings.TrimSpace(v) != "" {
		return v
	}
	if v, ok := os.LookupEnv(legacyKey); ok {
		warnLegacyEnvOnce(legacyKey, adstackKey)
		return v
	}
	return ""
}

func LicenseEnv(suffix string) string {
	return envStringDual("ADSTACK_LICENSE_"+suffix, "ESPX_LICENSE_"+suffix)
}

func BPFEnv(suffix string) string {
	return envStringDual("ADSTACK_BPF_"+suffix, "ESPX_BPF_"+suffix)
}

func RegionCodeFromEnv() int {
	if v, ok := os.LookupEnv("ADSTACK_REGION_CODE"); ok && strings.TrimSpace(v) != "" {
		return atoiEnv("ADSTACK_REGION_CODE", 0)
	}
	if _, ok := os.LookupEnv("ESPX_REGION_CODE"); ok {
		warnLegacyEnvOnce("ESPX_REGION_CODE", "ADSTACK_REGION_CODE")
		return atoiEnv("ESPX_REGION_CODE", 0)
	}
	return 0
}

func TelemetryOptInFromEnvDual() bool {
	if v, ok := os.LookupEnv("ADSTACK_TELEMETRY_OPT_IN"); ok {
		return parseBoolEnv(v)
	}
	if v, ok := os.LookupEnv("ESPX_TELEMETRY_OPT_IN"); ok {
		warnLegacyEnvOnce("ESPX_TELEMETRY_OPT_IN", "ADSTACK_TELEMETRY_OPT_IN")
		return parseBoolEnv(v)
	}
	return false
}

func TelemetryURLFromEnv() string {
	return envStringDual("ADSTACK_TELEMETRY_URL", "ESPX_TELEMETRY_URL")
}

func TelemetryIntervalSecFromEnv() int {
	if v, ok := os.LookupEnv("ADSTACK_TELEMETRY_INTERVAL_SEC"); ok && strings.TrimSpace(v) != "" {
		return atoiEnv("ADSTACK_TELEMETRY_INTERVAL_SEC", 3600)
	}
	if _, ok := os.LookupEnv("ESPX_TELEMETRY_INTERVAL_SEC"); ok {
		warnLegacyEnvOnce("ESPX_TELEMETRY_INTERVAL_SEC", "ADSTACK_TELEMETRY_INTERVAL_SEC")
		return atoiEnv("ESPX_TELEMETRY_INTERVAL_SEC", 3600)
	}
	return 3600
}

func TelemetryHTTPTimeoutSecFromEnv() int {
	if v, ok := os.LookupEnv("ADSTACK_TELEMETRY_TIMEOUT_SEC"); ok && strings.TrimSpace(v) != "" {
		return atoiEnv("ADSTACK_TELEMETRY_TIMEOUT_SEC", 5)
	}
	if _, ok := os.LookupEnv("ESPX_TELEMETRY_TIMEOUT_SEC"); ok {
		warnLegacyEnvOnce("ESPX_TELEMETRY_TIMEOUT_SEC", "ADSTACK_TELEMETRY_TIMEOUT_SEC")
		return atoiEnv("ESPX_TELEMETRY_TIMEOUT_SEC", 5)
	}
	return 5
}

func atoiEnv(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := parseInt(v); err == nil {
			return n
		}
	}
	return fallback
}

func parseBoolEnv(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseInt(raw string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(raw))
}
