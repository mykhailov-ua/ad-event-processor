package ingestion

import (
	"context"
	"log/slog"
)

func (r *Registry) ReloadFullSnapshot(ctx context.Context) (int, error) {
	count, err := r.Sync(ctx)
	if err != nil {
		return count, err
	}

	r.mu.Lock()
	w := r.budgetWarmer
	r.mu.Unlock()

	if w != nil {
		if warmed, warmErr := w.WarmFromRegistry(ctx, r); warmErr != nil {
			slog.Warn("registry full sync: budget warm failed", "error", warmErr)
		} else if warmed > 0 {
			slog.Debug("registry full sync: budget keys warmed", "keys", warmed)
		}
	}

	slog.Info("campaign registry full sync", "campaigns", count)
	return count, nil
}
