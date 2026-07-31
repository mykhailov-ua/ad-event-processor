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
	api, closeFn, err := notifier.OpenAPIOrDial(ctx, cfg)
	if err != nil {
		return nil, func() {}, err
	}
	if api == nil {
		return nil, func() {}, nil
	}
	return NewNotifierClientFromAPI(api), closeFn, nil
}
