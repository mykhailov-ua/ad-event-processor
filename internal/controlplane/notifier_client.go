package controlplane

import (
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

func NewNotifierClientFromAPI(api notifier.NotifierAPI) *NotifierClient {
	if api == nil {
		return nil
	}
	return &NotifierClient{api: api}
}

func NewNotifierClient(cfg *config.Config) (*NotifierClient, error) {
	if cfg == nil || !cfg.NotifierDialEnabled() {
		return nil, nil
	}

	target := cfg.Notifier.ServerHost + ":" + cfg.Notifier.Port
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("notifier gRPC dial %s: %w", target, err)
	}

	pbClient := notifierpb.NewNotifierServiceClient(conn)
	return &NotifierClient{
		conn: conn,
		api:  notifier.NewGRPCNotifierAPI(pbClient),
	}, nil
}

func (client *NotifierClient) API() notifier.NotifierAPI {
	if client == nil {
		return nil
	}
	return client.api
}

func (client *NotifierClient) Close() error {
	if client == nil || client.conn == nil {
		return nil
	}
	return client.conn.Close()
}
