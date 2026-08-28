package opsadmin

import (
	"strings"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"
)

func ResolveOpsAlertTarget(cfg *config.Config) (string, string, bool) {
	return notify.ResolveOpsAlertTarget(cfg)
}

func ResolveOpsAlertTargets(cfg *config.Config) []notify.OpsAlertTarget {
	return notify.ResolveOpsAlertTargets(cfg)
}

func ResolveBroadcastProviders(cfg *config.Config) []string {
	return notify.ResolveBroadcastProviders(cfg)
}

func AlertSeverityBroadcast(alert AlertmanagerAlert) bool {
	severity := strings.ToLower(strings.TrimSpace(alert.Labels["severity"]))
	return severity == "critical"
}
