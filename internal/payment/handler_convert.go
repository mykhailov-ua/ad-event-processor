package payment

import (
	"espx/internal/payment/db"
	"espx/internal/payment/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapStatusToPB(s db.PaymentPaymentIntentStatus) pb.PaymentIntentStatus {
	switch s {
	case db.PaymentPaymentIntentStatusCREATED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_CREATED
	case db.PaymentPaymentIntentStatusPENDINGPROVIDER:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_PENDING_PROVIDER
	case db.PaymentPaymentIntentStatusPROCESSING:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_PROCESSING
	case db.PaymentPaymentIntentStatusSUCCEEDED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_SUCCEEDED
	case db.PaymentPaymentIntentStatusFAILED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_FAILED
	case db.PaymentPaymentIntentStatusCANCELLED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_CANCELLED
	case db.PaymentPaymentIntentStatusREFUNDED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_REFUNDED
	case db.PaymentPaymentIntentStatusSETTLEMENTFAILED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_SETTLEMENT_FAILED
	case db.PaymentPaymentIntentStatusDISPUTED:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_DISPUTED
	default:
		return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_UNSPECIFIED
	}
}

func intentStatusString(s db.PaymentPaymentIntentStatus) string {
	return mapStatusToPB(s).String()
}

func intentToPB(intent db.PaymentPaymentIntent) *pb.PaymentIntent {
	return &pb.PaymentIntent{
		Id:             uuid.UUID(intent.ID.Bytes).String(),
		CustomerId:     uuid.UUID(intent.CustomerID.Bytes).String(),
		AmountMicro:    intent.AmountMicro,
		Currency:       intent.Currency,
		Status:         mapStatusToPB(intent.Status),
		Provider:       intent.Provider,
		ProviderRef:    intent.ProviderRef.String,
		IdempotencyKey: intent.IdempotencyKey,
		CreatedAt:      timestamppb.New(intent.CreatedAt.Time),
		UpdatedAt:      timestamppb.New(intent.UpdatedAt.Time),
	}
}
