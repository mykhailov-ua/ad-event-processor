package controlplane

import (
	"context"

	"espx/internal/config"
	"espx/internal/notifier"
)

func TryNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, func(), error) {
	if cfg == nil || !cfg.NotifierDialEnabled() {
		return nil, func() {}, nil
	}
	if !cfg.NotifierGRPCEnabled {
		mod, err := notifier.OpenModule(ctx, cfg)
		if err != nil {
			return nil, func() {}, err
		}
		if mod == nil {
			return nil, func() {}, nil
		}
		client := NewNotifierClientFromAPI(mod.API())
		return client, mod.Close, nil
	}
	client, err := NewNotifierClient(cfg)
	if err != nil {
		return nil, func() {}, err
	}
	return client, func() {
		if client != nil {
			_ = client.Close()
		}
	}, nil
}
