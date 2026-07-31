package controlplane

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

func (client *NotifierClient) SendNotification(ctx context.Context, provider notifierpb.Provider, recipient, title, body string) (*notifierpb.SendNotificationResponse, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("notifier client not configured")
	}
	result, err := client.api.SendNotification(ctx, provider.String(), recipient, title, body)
	if err != nil {
		return nil, err
	}
	return &notifierpb.SendNotificationResponse{
		NotificationId: result.NotificationID,
		Deduplicated:   result.Deduplicated,
	}, nil
}

func (client *NotifierClient) SendNotificationBatch(ctx context.Context, notifications []*notifierpb.SendNotificationRequest) (*notifierpb.SendNotificationBatchResponse, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("notifier client not configured")
	}
	inputs := make([]notifier.NotificationInput, 0, len(notifications))
	for _, item := range notifications {
		inputs = append(inputs, notifier.NotificationInputFromPB(item))
	}
	results, err := client.api.SendNotificationBatch(ctx, inputs)
	if err != nil {
		return nil, err
	}
	out := make([]*notifierpb.SendNotificationResponse, 0, len(results))
	for _, item := range results {
		out = append(out, &notifierpb.SendNotificationResponse{
			NotificationId: item.NotificationID,
			Deduplicated:   item.Deduplicated,
		})
	}
	return &notifierpb.SendNotificationBatchResponse{Notifications: out}, nil
}

func (client *NotifierClient) SendBroadcastNotification(
	ctx context.Context,
	provider notifierpb.Provider,
	recipient, title, body string,
	broadcastProviders []notifierpb.Provider,
) (*notifierpb.SendNotificationResponse, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("notifier client not configured")
	}
	broadcastProviderNames := make([]string, 0, len(broadcastProviders))
	for _, p := range broadcastProviders {
		broadcastProviderNames = append(broadcastProviderNames, p.String())
	}
	result, err := client.api.SendNotificationInput(ctx, notifier.NotificationInput{
		Provider:           provider.String(),
		Recipient:          recipient,
		Title:              title,
		Body:               body,
		Broadcast:          true,
		BroadcastProviders: broadcastProviderNames,
	})
	if err != nil {
		return nil, err
	}
	return &notifierpb.SendNotificationResponse{
		NotificationId: result.NotificationID,
		Deduplicated:   result.Deduplicated,
	}, nil
}
