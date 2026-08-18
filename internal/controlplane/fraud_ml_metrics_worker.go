package controlplane

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"
)

const fraudMLEvalMetricsInterval = 2 * time.Minute

type MLEvalMetricsWorker struct {
	svc *Service
}

func NewMLEvalMetricsWorker(svc *Service) *MLEvalMetricsWorker {
	return &MLEvalMetricsWorker{svc: svc}
}

func (w *MLEvalMetricsWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	ticker := time.NewTicker(fraudMLEvalMetricsInterval)
	defer ticker.Stop()
	w.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.refresh(ctx)
		}
	}
}

func (w *MLEvalMetricsWorker) refresh(ctx context.Context) {
	report := w.svc.loadFraudEvalReport(ctx, time.Now().UTC())
	publishFraudMLEvalMetrics(report)
}

func publishFraudMLEvalMetrics(report fraudEvalReport) {
	if !report.Available {
		metrics.FraudMLShadowPrecision.Set(0)
		metrics.FraudMLShadowRecall.Set(0)
		metrics.FraudMLDriftDetected.Set(0)
		metrics.FraudMLEvalGeneratedAtTimestamp.Set(0)
		return
	}
	metrics.FraudMLShadowPrecision.Set(report.Precision)
	metrics.FraudMLShadowRecall.Set(report.Recall)
	if report.DriftDetected {
		metrics.FraudMLDriftDetected.Set(1)
	} else {
		metrics.FraudMLDriftDetected.Set(0)
	}
	if !report.GeneratedTime.IsZero() {
		metrics.FraudMLEvalGeneratedAtTimestamp.Set(float64(report.GeneratedTime.Unix()))
	} else {
		metrics.FraudMLEvalGeneratedAtTimestamp.Set(0)
	}
}
