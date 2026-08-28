package payment

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/notify"
)

type NotifierClient struct {
	closeFn func()
	api     notify.NotifierAPI
}

func NewInProcessNotifierClient(api notify.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func ResolveNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, func(), error) {
	if cfg == nil || !cfg.OpsAlertsEnabled() {
		return nil, func() {}, nil
	}
	api, closeFn, err := notify.OpenAPI(ctx, cfg)
	if err != nil {
		return nil, func() {}, err
	}
	if api == nil {
		return nil, closeFn, nil
	}
	client := &NotifierClient{api: api, closeFn: closeFn}
	return client, func() { _ = client.Close() }, nil
}

func (c *NotifierClient) Close() error {
	if c == nil || c.closeFn == nil {
		return nil
	}
	c.closeFn()
	c.closeFn = nil
	return nil
}
