package licensing

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	licenseStateGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ad_license_state",
		Help: "Deployment license state enum (1=ACTIVE 2=OFFLINE_WARN 3=OFFLINE_GRACE 4=GRACE 5=EXPIRED 6=REVOKED)",
	})
	licenseOfflineDaysGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ad_license_offline_days",
		Help: "Whole days since last successful license heartbeat while JWT remains valid",
	})
)

func init() {
	prometheus.MustRegister(licenseStateGauge, licenseOfflineDaysGauge)
}

func licenseStateEnumValue(state LicenseState) float64 {
	switch state {
	case StateActive:
		return 1
	case StateOfflineWarn:
		return 2
	case StateOfflineGrace:
		return 3
	case StateGrace:
		return 4
	case StateExpired:
		return 5
	case StateRevoked:
		return 6
	default:
		return 0
	}
}

func SetLicenseMetrics(state LicenseState, offlineDays int) {
	licenseStateGauge.Set(licenseStateEnumValue(state))
	if offlineDays < 0 {
		offlineDays = 0
	}
	licenseOfflineDaysGauge.Set(float64(offlineDays))
}
