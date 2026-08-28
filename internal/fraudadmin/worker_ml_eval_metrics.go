package fraudadmin

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const fraudMLEvalMetricsInterval = 2 * time.Minute

type MLEvalMetricsHost interface {
	Pool() *pgxpool.Pool
}

type MLEvalMetricsWorker struct {
	host MLEvalMetricsHost
}

func NewMLEvalMetricsWorker(host MLEvalMetricsHost) *MLEvalMetricsWorker {
	return &MLEvalMetricsWorker{host: host}
}

func (w *MLEvalMetricsWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil {
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
	RefreshFraudMLEvalMetrics(ctx, w.host.Pool(), time.Now().UTC())
}
