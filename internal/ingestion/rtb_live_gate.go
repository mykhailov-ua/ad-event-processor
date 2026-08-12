package ingestion

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	dto "github.com/prometheus/client_model/go"
)

const (
	rtbLiveGateMinParityRate   = 0.95
	rtbLiveGateMinShadowEvals  = 100
	rtbLiveGateDefaultWindow   = time.Hour
	rtbLiveGateMismatchReason  = "shadow_winner_mismatch_rate_high"
	rtbLiveGateReconcileReason = "budget_reconcile_divergence_high"
	rtbLiveGateInsufficient    = "insufficient_shadow_evals"
)

type RtbLiveGateResult struct {
	Ready         bool                     `json:"ready"`
	Reasons       []string                 `json:"reasons,omitempty"`
	Shadow        RtbShadowDiffSnapshotDTO `json:"shadow"`
	ReconcileHigh bool                     `json:"reconcile_high"`
}

func EvaluateRtbLiveGate(window time.Duration) RtbLiveGateResult {
	if window <= 0 {
		window = rtbLiveGateDefaultWindow
	}
	out := RtbLiveGateResult{
		Shadow: RtbShadowDiffForWindow(window),
	}
	out.ReconcileHigh = rtbBudgetReconcileHigh()
	var reasons []string
	if out.Shadow.ShadowEvals < rtbLiveGateMinShadowEvals {
		reasons = append(reasons, rtbLiveGateInsufficient)
	} else if out.Shadow.ParityRate < rtbLiveGateMinParityRate {
		reasons = append(reasons, rtbLiveGateMismatchReason)
	}
	if out.ReconcileHigh {
		reasons = append(reasons, rtbLiveGateReconcileReason)
	}
	out.Reasons = reasons
	out.Ready = len(reasons) == 0
	return out
}

func rtbBudgetReconcileHigh() bool {
	metric := &dto.Metric{}
	if err := metrics.RtbBudgetReconcileHigh.Write(metric); err != nil {
		return false
	}
	return metric.GetGauge().GetValue() >= 1
}

func CanEnableRtbLive(window time.Duration) (bool, []string) {
	gate := EvaluateRtbLiveGate(window)
	return gate.Ready, gate.Reasons
}
