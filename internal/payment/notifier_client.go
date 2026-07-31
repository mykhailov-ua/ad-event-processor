package payment

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/notifier"
	notifierpb "espx/internal/notifier/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NotifierClient struct {
	conn *grpc.ClientConn
	api  notifier.NotifierAPI
}

func NewInProcessNotifierClient(api notifier.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func NewNotifierClient(cfg *config.Config) (*NotifierClient, error) {
	if cfg == nil || !cfg.OpsAlertsEnabled() {
		return nil, nil
	}

	target := cfg.Notifier.ServerHost + ":" + cfg.Notifier.Port
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("notifier gRPC dial %s: %w", target, err)
	}

	return &NotifierClient{
		conn: conn,
		api:  notifier.NewGRPCNotifierAPI(notifierpb.NewNotifierServiceClient(conn)),
	}, nil
}

func ResolveNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, func(), error) {
	if cfg == nil || !cfg.OpsAlertsEnabled() {
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
		return NewInProcessNotifierClient(mod.API()), mod.Close, nil
	}
	client, err := NewNotifierClient(cfg)
	if err != nil {
		return nil, func() {}, err
	}
	closeFn := func() {}
	if client != nil {
		closeFn = func() { _ = client.Close() }
	}
	return client, closeFn, nil
}

func (client *NotifierClient) Close() error {
	if client == nil || client.conn == nil {
		return nil
	}
	return client.conn.Close()
}
