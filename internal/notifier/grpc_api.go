package notifier

import (
	"context"

	"espx/internal/notifier/pb"
)

type grpcNotifierAPI struct {
	client pb.NotifierServiceClient
}

func NewGRPCNotifierAPI(client pb.NotifierServiceClient) NotifierAPI {
	if client == nil {
		return nil
	}
	return &grpcNotifierAPI{client: client}
}

func (g *grpcNotifierAPI) SendNotification(ctx context.Context, provider, recipient, title, body string) (SendNotificationResult, error) {
	return g.SendNotificationInput(ctx, NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (g *grpcNotifierAPI) SendNotificationInput(ctx context.Context, input NotificationInput) (SendNotificationResult, error) {
	req, err := input.toPB()
	if err != nil {
		return SendNotificationResult{}, err
	}
	resp, err := g.client.SendNotification(ctx, req)
	if err != nil {
		return SendNotificationResult{}, err
	}
	return SendNotificationResult{
		NotificationID: resp.NotificationId,
		Deduplicated:   resp.Deduplicated,
	}, nil
}

func (g *grpcNotifierAPI) SendNotificationBatch(ctx context.Context, inputs []NotificationInput) ([]SendNotificationResult, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	reqs := make([]*pb.SendNotificationRequest, 0, len(inputs))
	for _, item := range inputs {
		req, err := item.toPB()
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	resp, err := g.client.SendNotificationBatch(ctx, &pb.SendNotificationBatchRequest{Notifications: reqs})
	if err != nil {
		return nil, err
	}
	out := make([]SendNotificationResult, 0, len(resp.Notifications))
	for _, item := range resp.Notifications {
		out = append(out, SendNotificationResult{
			NotificationID: item.NotificationId,
			Deduplicated:   item.Deduplicated,
		})
	}
	return out, nil
}
