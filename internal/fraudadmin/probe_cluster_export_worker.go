package fraudadmin

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/pkg/probecluster"
)

type ProbeClusterExportWorker struct {
	host     ProbeClusterHost
	export   *ProbeCluster
	policy   probecluster.Policy
	interval time.Duration
}

func NewProbeClusterExportWorker(host ProbeClusterHost, export *ProbeCluster, policy probecluster.Policy, interval time.Duration) *ProbeClusterExportWorker {
	if host == nil || export == nil || interval <= 0 {
		return nil
	}
	return &ProbeClusterExportWorker{
		host:     host,
		export:   export,
		policy:   policy,
		interval: interval,
	}
}

func (w *ProbeClusterExportWorker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("probe cluster export worker started", "interval", w.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			exported, err := w.export.ExportHotClusters(ctx, w.policy)
			if err != nil {
				slog.Warn("probe cluster export failed", "error", err)
				continue
			}
			if exported > 0 {
				slog.Info("probe cluster export completed", "exported", exported)
			}
		}
	}
}
