package domain

import "espx/internal/config"

const DefaultRtbCatalogReloadChannel = "rtb:catalog:reload"

func RtbCatalogReloadChannel(cfg *config.Config) string {
	if cfg != nil && cfg.RtbCatalogReloadChannel != "" {
		return cfg.RtbCatalogReloadChannel
	}
	return DefaultRtbCatalogReloadChannel
}
