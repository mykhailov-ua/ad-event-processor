package config

import "os"

func LicenseRequiredFromEnv() bool {
	if v := LicenseEnv("REQUIRED"); v != "" {
		return parseBoolEnv(v)
	}
	return os.Getenv("ESPX_PROFILE") == "production"
}

func LicenseFileRecheckInterval() string {
	if v := LicenseEnv("FILE_RECHECK_INTERVAL"); v != "" {
		return v
	}
	return "5m"
}
