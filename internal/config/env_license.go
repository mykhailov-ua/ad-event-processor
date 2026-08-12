package config

import (
	"os"
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

func LicenseFileRecheckInterval() string {
	if v := LicenseEnv("FILE_RECHECK_INTERVAL"); v != "" {
		return v
	}
	return "5m"
}
