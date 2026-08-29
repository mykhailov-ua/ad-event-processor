package checkout

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/payment/db"

	"github.com/google/uuid"
)

type createPaymentIntentResult = CreateIntentResult

func paymentIntentFromDB(intent db.PaymentPaymentIntent) domain.PaymentIntent {
	out := domain.PaymentIntent{
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

func disputeFromListItem(item DisputeListItem) domain.Dispute {
	out := domain.Dispute{
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

func paymentIntentStatusString(s db.PaymentPaymentIntentStatus) string {
	switch s {
	case db.PaymentPaymentIntentStatusCREATED:
		return "PAYMENT_INTENT_STATUS_CREATED"
	case db.PaymentPaymentIntentStatusPENDINGPROVIDER:
		return "PAYMENT_INTENT_STATUS_PENDING_PROVIDER"
	case db.PaymentPaymentIntentStatusPROCESSING:
		return "PAYMENT_INTENT_STATUS_PROCESSING"
	case db.PaymentPaymentIntentStatusSUCCEEDED:
		return "PAYMENT_INTENT_STATUS_SUCCEEDED"
	case db.PaymentPaymentIntentStatusFAILED:
		return "PAYMENT_INTENT_STATUS_FAILED"
	case db.PaymentPaymentIntentStatusCANCELLED:
		return "PAYMENT_INTENT_STATUS_CANCELLED"
	case db.PaymentPaymentIntentStatusREFUNDED:
		return "PAYMENT_INTENT_STATUS_REFUNDED"
	case db.PaymentPaymentIntentStatusSETTLEMENTFAILED:
		return "PAYMENT_INTENT_STATUS_SETTLEMENT_FAILED"
	case db.PaymentPaymentIntentStatusDISPUTED:
		return "PAYMENT_INTENT_STATUS_DISPUTED"
	default:
		return "PAYMENT_INTENT_STATUS_UNSPECIFIED"
	}
}
