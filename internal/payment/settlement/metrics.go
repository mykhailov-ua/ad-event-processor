package settlement

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OutboxPending = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "payment_outbox_pending",
		Help: "Payment outbox rows awaiting settlement",
	})

	SettlementErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payment_settlement_errors_total",
		Help: "Failed settlement attempts from the payment outbox worker",
	})

	FinancialReconRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_financial_recon_runs_total",
		Help: "Payment financial reconciliation runs by terminal status",
	}, []string{"status"})

	FinancialReconFindingsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_financial_recon_findings_total",
		Help: "Payment financial reconciliation findings persisted by kind",
	}, []string{"kind"})
)
