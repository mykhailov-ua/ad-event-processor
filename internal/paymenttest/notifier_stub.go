package paymenttest

import (
	"context"
	"sync"

	"espx/internal/config"
	notifierpb "espx/internal/notifier/pb"

	"google.golang.org/grpc"
)

type StubNotifierClient struct {
	mu       sync.Mutex
	requests []*notifierpb.SendNotificationRequest
}

func (stub *StubNotifierClient) SendNotification(
	_ context.Context,
	in *notifierpb.SendNotificationRequest,
	_ ...grpc.CallOption,
) (*notifierpb.SendNotificationResponse, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.requests = append(stub.requests, in)
	return &notifierpb.SendNotificationResponse{NotificationId: "stub-id"}, nil
}

func (stub *StubNotifierClient) SendNotificationBatch(
	_ context.Context,
	in *notifierpb.SendNotificationBatchRequest,
	_ ...grpc.CallOption,
) (*notifierpb.SendNotificationBatchResponse, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.requests = append(stub.requests, in.Notifications...)
	return &notifierpb.SendNotificationBatchResponse{}, nil
}

func (stub *StubNotifierClient) GetNotification(
	_ context.Context,
	_ *notifierpb.GetNotificationRequest,
	_ ...grpc.CallOption,
) (*notifierpb.GetNotificationResponse, error) {
	return nil, nil
}

func (stub *StubNotifierClient) Snapshot() []*notifierpb.SendNotificationRequest {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	out := make([]*notifierpb.SendNotificationRequest, len(stub.requests))
	copy(out, stub.requests)
	return out
}

func TestOpsConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true
	cfg.Notifier.TelegramChatID = "-100123"
	cfg.Notifier.ServerHost = "127.0.0.1"
	cfg.Notifier.Port = "8085"
	return cfg
}
