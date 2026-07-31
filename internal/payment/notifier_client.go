package payment

import (
	"context"

	"espx/internal/config"
	"espx/internal/notifier"
)

type NotifierClient struct {
	closeFn func()
	api     notifier.NotifierAPI
}

func NewInProcessNotifierClient(api notifier.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func ResolveNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, func(), error) {
	if cfg == nil || !cfg.OpsAlertsEnabled() {
		return nil, func() {}, nil
	}
	api, closeFn, err := notifier.OpenAPIOrDial(ctx, cfg)
	if err != nil {
		return nil, func() {}, err
	}
	if api == nil {
		return nil, closeFn, nil
	}
	client := &NotifierClient{api: api, closeFn: closeFn}
	return client, func() { _ = client.Close() }, nil
}

func (client *NotifierClient) Close() error {
	if client == nil || client.closeFn == nil {
		return nil
	}
	client.closeFn()
	client.closeFn = nil
	return nil
}
