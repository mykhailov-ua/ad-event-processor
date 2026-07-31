package controlplane

import (
	"context"

	"espx/internal/config"
	"espx/internal/notifier"
)

func NewNotifierClientInProcess(api notifier.NotifierAPI) *NotifierClient {
	return NewNotifierClientFromAPI(api)
}

func openNotifierClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*NotifierClient, func(), error) {
	if opts.Notifier != nil {
		return opts.Notifier, func() {}, nil
	}
	return TryNotifierClient(ctx, cfg)
}
