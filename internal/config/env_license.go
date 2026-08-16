package config

import (
	"os"
	"strings"
	"time"
)

func LicenseRequiredFromEnv() bool {
	if v := LicenseEnv("REQUIRED"); v != "" {
		return parseBoolEnv(v)
	}
	return ProfileFromEnv() == "production"
}

func LicensePathFromEnv() string {
	if v := LicenseEnv("PATH"); v != "" {
		return v
	}
	return "license.jwt"
}

func LicenseFilePresent() bool {
	path := LicensePathFromEnv()
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Size() > 0
}

func LicenseProbeEnabled() bool {
	return LicenseRequiredFromEnv() || LicenseFilePresent()
}

// License mode and sealed asset policy (cold path).
func LicenseMode() string {
	return strings.ToLower(strings.TrimSpace(LicenseEnv("MODE")))
}

func LicenseAssetsUnsealed() bool {
	mode := LicenseMode()
	if mode == "dev" || mode == "development" {
		return true
	}
	if v := LicenseEnv("ASSET_SEAL"); v != "" {
		return !parseBoolEnv(v)
	}
	return mode == ""
}

func LicenseSeedCouplingEnabled() bool {
	if LicenseAssetsUnsealed() {
		return false
	}
	if v := LicenseEnv("SEED_COUPLE"); v != "" {
		return parseBoolEnv(v)
	}
	return LicenseMode() == "enterprise" || LicenseMode() == "file"
}

func LicenseFileRecheckInterval() string {
	if v := LicenseEnv("FILE_RECHECK_INTERVAL"); v != "" {
		return v
	}
	return "5m"
}

func LicenseSkewWatchEnabled() bool {
	if LicenseMode() == "dev" || LicenseMode() == "development" {
		return false
	}
	if v := LicenseEnv("SKEW_WATCH"); v != "" {
		return parseBoolEnv(v)
	}
	return LicenseRequiredFromEnv() || LicenseFilePresent()
}

func LicenseSkewWatchInterval() time.Duration {
	if v := LicenseEnv("SKEW_WATCH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return time.Hour
}

func LicenseSkewWatchThreshold() time.Duration {
	if v := LicenseEnv("SKEW_WATCH_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 5 * time.Minute
}

func LicenseGuardEnvEnabled() bool {
	if v := LicenseEnv("GUARD"); v != "" {
		return parseBoolEnv(v)
	}
	return true
}

func LicenseGuardPtraceWatchdogEnabled() bool {
	if !LicenseGuardEnvEnabled() {
		return false
	}
	if v := LicenseEnv("GUARD_PTRACE"); v != "" {
		return parseBoolEnv(v)
	}
	return true
}
