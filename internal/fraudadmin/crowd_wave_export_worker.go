package fraudadmin

import (
	"context"
	"log/slog"
	"time"
)

type CrowdWaveExportWorker struct {
	host     CrowdWaveHost
	export   *CrowdWave
	interval time.Duration
}

func NewCrowdWaveExportWorker(host CrowdWaveHost, export *CrowdWave, interval time.Duration) *CrowdWaveExportWorker {
	if host == nil || export == nil || interval <= 0 {
		return nil
	}
	return &CrowdWaveExportWorker{
		host:     host,
		export:   export,
		interval: interval,
	}
}

func (w *CrowdWaveExportWorker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("crowd wave export worker started", "interval", w.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			exported, err := w.export.ExportActiveWaves(ctx)
			if err != nil {
				slog.Warn("crowd wave export failed", "error", err)
				continue
			}
			if exported > 0 {
				slog.Info("crowd wave export completed", "exported", exported)
			}
		}
	}
}
