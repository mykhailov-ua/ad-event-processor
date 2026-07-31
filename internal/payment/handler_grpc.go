package payment

import (
	"context"

	"espx/internal/payment/pb"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	intent := result.Intent
	return &pb.CreatePaymentIntentResponse{
		IntentId:    uuid.UUID(intent.ID.Bytes).String(),
		Status:      mapStatusToPB(intent.Status),
		CheckoutUrl: result.CheckoutURL,
		ProviderRef: intent.ProviderRef.String,
	}, nil
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
	return intentToPB(intent), nil
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
		Intents: coldpath.MapSlice(intents, intentToPB),
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
	items, total, err := handler.listDisputes(ctx, customerID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	disputes := make([]*pb.DisputeRecord, 0, len(items))
	for _, item := range items {
		rec := &pb.DisputeRecord{
			IntentId:          uuid.UUID(item.Intent.ID.Bytes).String(),
			CustomerId:        uuid.UUID(item.Intent.CustomerID.Bytes).String(),
			AmountMicro:       item.Intent.AmountMicro,
			Currency:          item.Intent.Currency,
			ProviderDisputeId: item.ProviderDisputeID,
		}
		if item.Intent.UpdatedAt.Valid {
			rec.UpdatedAt = timestamppb.New(item.Intent.UpdatedAt.Time)
		}
		disputes = append(disputes, rec)
	}
	return &pb.ListDisputesResponse{Disputes: disputes, Total: total}, nil
}

func (handler *Handler) ReplayWebhook(ctx context.Context, req *pb.ReplayWebhookRequest) (*pb.ReplayWebhookResponse, error) {
	statusText, err := handler.replayWebhook(ctx, req.Provider, req.ProviderEventId)
	if err != nil {
		return nil, err
	}
	return &pb.ReplayWebhookResponse{Status: statusText}, nil
}
