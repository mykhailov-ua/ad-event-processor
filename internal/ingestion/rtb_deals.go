package ingestion

import (
	"context"
	"log/slog"
	"time"

	"espx/internal/config"
	"espx/internal/domain"
	db "espx/internal/domain/db"

	"github.com/redis/go-redis/v9"
)

func ReloadRtbCatalog(
	ctx context.Context,
	q *db.Queries,
	registry *Registry,
	catalog *RtbCatalog,
	cfg *config.Config,
	hybrid *HybridBalancer,
	budgetSync RtbBudgetSync,
	watcher *SettingsWatcher,
) error {
	if err := domain.ReloadRtbDeals(ctx, q, catalog); err != nil {
		return err
	}
	if registry != nil && catalog != nil && cfg != nil && cfg.RtbEnabled() {
		SyncRtbCatalog(ctx, registry, catalog, cfg, hybrid, budgetSync, watcher)
		if allow, err := LoadSupplyChainAllowlist(ctx, q); err == nil {
			catalog.SetSupplyChainAllowlist(allow)
		}
	}
	return nil
}

func StartRtbCatalogReloadWatch(
	ctx context.Context,
	q *db.Queries,
	rdb redis.UniversalClient,
	channel string,
	registry *Registry,
	catalog *RtbCatalog,
	cfg *config.Config,
	hybrid *HybridBalancer,
	budgetSync RtbBudgetSync,
	watcher *SettingsWatcher,
) {
	if rdb == nil || catalog == nil || q == nil {
		return
	}
	if channel == "" {
		channel = domain.DefaultRtbCatalogReloadChannel
	}

	reload := func() {
		reloadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := ReloadRtbCatalog(reloadCtx, q, registry, catalog, cfg, hybrid, budgetSync, watcher); err != nil {
			slog.Error("rtb catalog reload failed", "error", err)
			return
		}
		slog.Info("rtb catalog reloaded via pubsub", "deals", catalog.DealCount())
	}

	go func() {
		pubsub := rdb.Subscribe(ctx, channel)
		defer pubsub.Close()

		ch := pubsub.Channel(redis.WithChannelSize(64))
		trigger := make(chan struct{}, 1)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-trigger:
					reload()
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					slog.Error("rtb catalog reload pubsub channel closed")
					return
				}
				if msg == nil {
					continue
				}
				select {
				case trigger <- struct{}{}:
				default:
				}
			}
		}
	}()
}
