package management

import (
	"context"
	"fmt"

	"espx/internal/config"
	notifierpb "espx/internal/notifier/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NotifierClient struct {
	conn   *grpc.ClientConn
	client notifierpb.NotifierServiceClient
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

	return &NotifierClient{
		conn:   conn,
		client: notifierpb.NewNotifierServiceClient(conn),
	}, nil
}

func (client *NotifierClient) Close() error {
	if client == nil || client.conn == nil {
		return nil
	}
	return client.conn.Close()
}

func (client *NotifierClient) SendNotification(ctx context.Context, provider notifierpb.Provider, recipient, title, body string) (*notifierpb.SendNotificationResponse, error) {
	if client == nil || client.client == nil {
		return nil, fmt.Errorf("notifier client not configured")
	}
	return client.client.SendNotification(ctx, &notifierpb.SendNotificationRequest{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (client *NotifierClient) SendNotificationBatch(ctx context.Context, notifications []*notifierpb.SendNotificationRequest) (*notifierpb.SendNotificationBatchResponse, error) {
	if client == nil || client.client == nil {
		return nil, fmt.Errorf("notifier client not configured")
	}
	return client.client.SendNotificationBatch(ctx, &notifierpb.SendNotificationBatchRequest{
		Notifications: notifications,
	})
}

func (client *NotifierClient) SendBroadcastNotification(
	ctx context.Context,
	provider notifierpb.Provider,
	recipient, title, body string,
	broadcastProviders []notifierpb.Provider,
) (*notifierpb.SendNotificationResponse, error) {
	if client == nil || client.client == nil {
		return nil, fmt.Errorf("notifier client not configured")
	}
	return client.client.SendNotification(ctx, &notifierpb.SendNotificationRequest{
		Provider:           provider,
		Recipient:          recipient,
		Title:              title,
		Body:               body,
		DeliveryMode:       notifierpb.DeliveryMode_DELIVERY_MODE_BROADCAST,
		BroadcastProviders: broadcastProviders,
	})
}
