package controlplane

import (
	"context"

	"espx/internal/config"
	"espx/internal/notifier"
)

type NotifierClient struct {
	closeFn func()
	api     notifier.NotifierAPI
}

func NewNotifierClientFromAPI(api notifier.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func NewNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, error) {
	if cfg == nil || !cfg.NotifierDialEnabled() {
		return nil, nil
	}
	api, closeFn, err := notifier.OpenAPIOrDial(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if api == nil {
		return nil, nil
	}
	return &NotifierClient{api: api, closeFn: closeFn}, nil
}

func NewNotifierClientInProcess(api notifier.NotifierAPI) *NotifierClient {
	return NewNotifierClientFromAPI(api)
}

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

func openNotifierClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*NotifierClient, func(), error) {
	if opts.Notifier != nil {
		return opts.Notifier, func() {}, nil
	}
	return TryNotifierClient(ctx, cfg)
}

func (client *NotifierClient) API() notifier.NotifierAPI {
	if client == nil {
		return nil
	}
	return client.api
}

func (client *NotifierClient) Close() error {
	if client == nil || client.closeFn == nil {
		return nil
	}
	client.closeFn()
	client.closeFn = nil
	return nil
}
