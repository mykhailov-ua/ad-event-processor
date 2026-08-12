package metrics

func SetControlOutboxQueueMetrics(pending int64, oldestSeconds float64) {
	p := float64(pending)
	ControlOutboxPendingTotal.Set(p)
	ManagementOutboxPendingTotal.Set(p)
	ControlOutboxOldestPendingSeconds.Set(oldestSeconds)
	ManagementOutboxOldestPendingSeconds.Set(oldestSeconds)
}

func IncControlOpsAlertEnqueueFailures() {
	ControlOpsAlertEnqueueFailuresTotal.Inc()
	ManagementOpsAlertEnqueueFailuresTotal.Inc()
}

func AddControlOpsAlertEnqueueFailures(n float64) {
	ControlOpsAlertEnqueueFailuresTotal.Add(n)
	ManagementOpsAlertEnqueueFailuresTotal.Add(n)
}

func AddControlCommissionsCollected(v float64) {
	ControlCommissionsCollectedTotal.Add(v)
	CommissionsCollectedTotal.Add(v)
}

func AddControlBalanceTopup(currency string, v float64) {
	ControlBalanceTopupsTotal.WithLabelValues(currency).Add(v)
	BalanceTopupsTotal.WithLabelValues(currency).Add(v)
}

func SetControlActiveCampaigns(n float64) {
	ControlActiveCampaigns.Set(n)
	ActiveCampaigns.Set(n)
}
