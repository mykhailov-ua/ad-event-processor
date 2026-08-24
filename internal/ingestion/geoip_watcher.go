package ingestion

import (
	"context"
	"log/slog"
	"os"
	"time"

	"ad-event-processor/internal/metrics"
)

type GeoIPWatcher struct {
	provider    *MaxMindProvider
	countryPath string
	asnPath     string
	interval    time.Duration
}

func NewGeoIPWatcher(provider *MaxMindProvider, countryDBPath, asnDBPath string, interval time.Duration) *GeoIPWatcher {
	if interval <= 0 {
		interval = time.Minute
	}
	return &GeoIPWatcher{
		provider:    provider,
		countryPath: countryDBPath,
		asnPath:     asnDBPath,
		interval:    interval,
	}
}

func (w *GeoIPWatcher) Start(ctx context.Context) {
	if w == nil || w.provider == nil {
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var lastCountryMod, lastASNMod time.Time
	if info, err := os.Stat(w.countryPath); err == nil {
		lastCountryMod = info.ModTime()
	}
	if w.asnPath != "" {
		if info, err := os.Stat(w.asnPath); err == nil {
			lastASNMod = info.ModTime()
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.countryPath != "" {
				info, err := os.Stat(w.countryPath)
				if err != nil {
					slog.Debug("geoip watcher stat failed", "path", w.countryPath, "error", err)
				} else if info.ModTime().After(lastCountryMod) {
					if err := w.provider.Reload(w.countryPath); err != nil {
						metrics.GeoIPReloadErrorsTotal.Inc()
						slog.Warn("geoip hot reload failed", "path", w.countryPath, "error", err)
					} else {
						lastCountryMod = info.ModTime()
						slog.Info("geoip database hot-reloaded", "path", w.countryPath, "mtime", lastCountryMod.UTC().Format(time.RFC3339))
					}
				}
			}
			if w.asnPath != "" {
				info, err := os.Stat(w.asnPath)
				if err != nil {
					slog.Debug("geoip asn watcher stat failed", "path", w.asnPath, "error", err)
				} else if info.ModTime().After(lastASNMod) {
					if err := w.provider.ReloadASN(w.asnPath); err != nil {
						metrics.GeoIPReloadErrorsTotal.Inc()
						slog.Warn("geoip asn hot reload failed", "path", w.asnPath, "error", err)
					} else {
						lastASNMod = info.ModTime()
						slog.Info("geoip asn database hot-reloaded", "path", w.asnPath, "mtime", lastASNMod.UTC().Format(time.RFC3339))
					}
				}
			}
		}
	}
}
