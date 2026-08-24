package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"ad-event-processor/pkg/naming"
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
	adEventKey := "AD_EVENT_PROCESSOR_LICENSE_" + suffix
	if v, ok := os.LookupEnv(adEventKey); ok && strings.TrimSpace(v) != "" {
		return v
	}
	legacyKey := naming.LegacyVendorEnvKey("LICENSE_" + suffix)
	if v, ok := os.LookupEnv(legacyKey); ok {
		warnLegacyEnvOnce(legacyKey, adEventKey)
		return v
	}
	adstackKey := "ADSTACK_LICENSE_" + suffix
	if v, ok := os.LookupEnv(adstackKey); ok {
		warnLegacyEnvOnce(adstackKey, adEventKey)
		return v
	}
	return ""
}

func InstallRootFromEnv() string {
	return envStringDual("AD_EVENT_PROCESSOR_INSTALL_ROOT", naming.LegacyVendorEnvKey("INSTALL_ROOT"))
}

func ProfileFromEnv() string {
	return envStringDual("AD_EVENT_PROCESSOR_PROFILE", naming.LegacyVendorEnvKey("PROFILE"))
}

func BPFEnv(suffix string) string {
	return envStringDual("ADSTACK_BPF_"+suffix, naming.LegacyVendorEnvKey("BPF_"+suffix))
}

func RegionCodeFromEnv() int {
	if v, ok := os.LookupEnv("ADSTACK_REGION_CODE"); ok && strings.TrimSpace(v) != "" {
		return atoiEnv("ADSTACK_REGION_CODE", 0)
	}
	legacyKey := naming.LegacyVendorEnvKey("REGION_CODE")
	if _, ok := os.LookupEnv(legacyKey); ok {
		warnLegacyEnvOnce(legacyKey, "ADSTACK_REGION_CODE")
		return atoiEnv(legacyKey, 0)
	}
	return 0
}

func TelemetryOptInFromEnvDual() bool {
	if v, ok := os.LookupEnv("ADSTACK_TELEMETRY_OPT_IN"); ok {
		return parseBoolEnv(v)
	}
	legacyKey := naming.LegacyVendorEnvKey("TELEMETRY_OPT_IN")
	if v, ok := os.LookupEnv(legacyKey); ok {
		warnLegacyEnvOnce(legacyKey, "ADSTACK_TELEMETRY_OPT_IN")
		return parseBoolEnv(v)
	}
	return false
}

func TelemetryURLFromEnv() string {
	return envStringDual("ADSTACK_TELEMETRY_URL", naming.LegacyVendorEnvKey("TELEMETRY_URL"))
}

func TelemetryIntervalSecFromEnv() int {
	if v, ok := os.LookupEnv("ADSTACK_TELEMETRY_INTERVAL_SEC"); ok && strings.TrimSpace(v) != "" {
		return atoiEnv("ADSTACK_TELEMETRY_INTERVAL_SEC", 3600)
	}
	legacyKey := naming.LegacyVendorEnvKey("TELEMETRY_INTERVAL_SEC")
	if _, ok := os.LookupEnv(legacyKey); ok {
		warnLegacyEnvOnce(legacyKey, "ADSTACK_TELEMETRY_INTERVAL_SEC")
		return atoiEnv(legacyKey, 3600)
	}
	return 3600
}

func TelemetryHTTPTimeoutSecFromEnv() int {
	if v, ok := os.LookupEnv("ADSTACK_TELEMETRY_TIMEOUT_SEC"); ok && strings.TrimSpace(v) != "" {
		return atoiEnv("ADSTACK_TELEMETRY_TIMEOUT_SEC", 5)
	}
	legacyKey := naming.LegacyVendorEnvKey("TELEMETRY_TIMEOUT_SEC")
	if _, ok := os.LookupEnv(legacyKey); ok {
		warnLegacyEnvOnce(legacyKey, "ADSTACK_TELEMETRY_TIMEOUT_SEC")
		return atoiEnv(legacyKey, 5)
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

func LegacyVendorEnvKey(suffix string) string {
	return naming.LegacyVendorEnvKey(suffix)
}

func LegacyIngressNativeSchema() string {
	return naming.DeprecatedIngressNativeSchema()
}
