package payment

import (
	"context"

	"espx/internal/payment/pb"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (handler *Handler) CreatePaymentIntent(ctx context.Context, req *pb.CreatePaymentIntentRequest) (*pb.CreatePaymentIntentResponse, error) {
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	result, err := handler.createPaymentIntent(ctx, customerID, req.AmountMicro, req.Currency, req.IdempotencyKey, req.Metadata)
	if err != nil {
		return nil, err
	}
	return createPaymentIntentResultToPB(result), nil
}

func (handler *Handler) GetPaymentIntent(ctx context.Context, req *pb.GetPaymentIntentRequest) (*pb.PaymentIntent, error) {
	intentID, err := uuid.Parse(req.IntentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid intent id")
	}
	intent, err := handler.getPaymentIntent(ctx, intentID)
	if err != nil {
		return nil, err
	}
	return PaymentIntentToPB(intent), nil
}

func (handler *Handler) ListPaymentIntents(ctx context.Context, req *pb.ListPaymentIntentsRequest) (*pb.ListPaymentIntentsResponse, error) {
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	intents, total, err := handler.listPaymentIntents(ctx, customerID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	return &pb.ListPaymentIntentsResponse{
		Intents: PaymentIntentsToPB(intents),
		Total:   total,
	}, nil
}

func (handler *Handler) ListDisputes(ctx context.Context, req *pb.ListDisputesRequest) (*pb.ListDisputesResponse, error) {
	var customerID *uuid.UUID
	if req.CustomerId != "" {
		parsed, err := uuid.Parse(req.CustomerId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid customer id")
		}
		customerID = &parsed
	}
	disputes, total, err := handler.listDisputes(ctx, customerID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	return &pb.ListDisputesResponse{
		Disputes: coldpath.MapSlice(disputes, DisputeToPB),
		Total:    total,
	}, nil
}

func (handler *Handler) ReplayWebhook(ctx context.Context, req *pb.ReplayWebhookRequest) (*pb.ReplayWebhookResponse, error) {
	statusText, err := handler.replayWebhook(ctx, req.Provider, req.ProviderEventId)
	if err != nil {
		return nil, err
	}
	return &pb.ReplayWebhookResponse{Status: statusText}, nil
}
