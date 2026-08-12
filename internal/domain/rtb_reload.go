package domain

import "github.com/bidshard/ad-event-processor/internal/config"

const DefaultRtbCatalogReloadChannel = "rtb:catalog:reload"

func RtbCatalogReloadChannel(cfg *config.Config) string {
	if cfg != nil && cfg.RtbCatalogReloadChannel != "" {
		return cfg.RtbCatalogReloadChannel
	}
	return DefaultRtbCatalogReloadChannel
}
