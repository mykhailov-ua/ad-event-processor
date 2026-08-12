package payment

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/notify"
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

func (client *NotifierClient) Close() error {
	if client == nil || client.closeFn == nil {
		return nil
	}
	client.closeFn()
	client.closeFn = nil
	return nil
}
