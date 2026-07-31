package payment

import (
	"espx/internal/payment/db"
	"espx/internal/payment/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createPaymentIntentResult struct {
	Intent      PaymentIntent
	CheckoutURL string
}

func paymentIntentFromDB(intent db.PaymentPaymentIntent) PaymentIntent {
	out := PaymentIntent{
		ID:             uuid.UUID(intent.ID.Bytes).String(),
		CustomerID:     uuid.UUID(intent.CustomerID.Bytes).String(),
		AmountMicro:    intent.AmountMicro,
		Currency:       intent.Currency,
		Status:         paymentIntentStatusString(intent.Status),
		Provider:       intent.Provider,
		IdempotencyKey: intent.IdempotencyKey,
	}
	if intent.ProviderRef.Valid {
		out.ProviderRef = intent.ProviderRef.String
	}
	if intent.CreatedAt.Valid {
		out.CreatedAt = intent.CreatedAt.Time.UTC()
	}
	if intent.UpdatedAt.Valid {
		out.UpdatedAt = intent.UpdatedAt.Time.UTC()
	}
	return out
}

func disputeFromListItem(item DisputeListItem) Dispute {
	out := Dispute{
		IntentID:          uuid.UUID(item.Intent.ID.Bytes).String(),
		CustomerID:        uuid.UUID(item.Intent.CustomerID.Bytes).String(),
		AmountMicro:       item.Intent.AmountMicro,
		Currency:          item.Intent.Currency,
		ProviderDisputeID: item.ProviderDisputeID,
	}
	if item.Intent.UpdatedAt.Valid {
		out.UpdatedAt = item.Intent.UpdatedAt.Time.UTC()
	}
	return out
}

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

func paymentIntentStatusString(s db.PaymentPaymentIntentStatus) string {
	return mapStatusToPB(s).String()
}

func paymentIntentStatusToPB(status string) pb.PaymentIntentStatus {
	if v, ok := pb.PaymentIntentStatus_value[status]; ok {
		return pb.PaymentIntentStatus(v)
	}
	return pb.PaymentIntentStatus_PAYMENT_INTENT_STATUS_UNSPECIFIED
}

func PaymentIntentToPB(intent PaymentIntent) *pb.PaymentIntent {
	out := &pb.PaymentIntent{
		Id:             intent.ID,
		CustomerId:     intent.CustomerID,
		AmountMicro:    intent.AmountMicro,
		Currency:       intent.Currency,
		Status:         paymentIntentStatusToPB(intent.Status),
		Provider:       intent.Provider,
		ProviderRef:    intent.ProviderRef,
		IdempotencyKey: intent.IdempotencyKey,
	}
	if !intent.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(intent.CreatedAt)
	}
	if !intent.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(intent.UpdatedAt)
	}
	return out
}

func PaymentIntentsToPB(intents []PaymentIntent) []*pb.PaymentIntent {
	if len(intents) == 0 {
		return nil
	}
	out := make([]*pb.PaymentIntent, 0, len(intents))
	for _, intent := range intents {
		out = append(out, PaymentIntentToPB(intent))
	}
	return out
}

func DisputeToPB(dispute Dispute) *pb.DisputeRecord {
	out := &pb.DisputeRecord{
		IntentId:          dispute.IntentID,
		CustomerId:        dispute.CustomerID,
		AmountMicro:       dispute.AmountMicro,
		Currency:          dispute.Currency,
		ProviderDisputeId: dispute.ProviderDisputeID,
	}
	if !dispute.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(dispute.UpdatedAt)
	}
	return out
}

func createPaymentIntentResultToPB(result createPaymentIntentResult) *pb.CreatePaymentIntentResponse {
	return &pb.CreatePaymentIntentResponse{
		IntentId:    result.Intent.ID,
		Status:      paymentIntentStatusToPB(result.Intent.Status),
		CheckoutUrl: result.CheckoutURL,
		ProviderRef: result.Intent.ProviderRef,
	}
}
