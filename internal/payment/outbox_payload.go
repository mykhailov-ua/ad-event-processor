package payment

import (
	"encoding/json"

	"github.com/google/uuid"
)

type settleBalanceOutboxPayload struct {
	CustomerID           string `json:"customer_id"`
	AmountMicro          int64  `json:"amount_micro"`
	LedgerIdempotencyKey string `json:"ledger_idempotency_key"`
	PaymentIntentID      string `json:"payment_intent_id"`
	Provider             string `json:"provider"`
	ProviderRef          string `json:"provider_ref"`
}

func marshalSettleBalanceOutbox(customerID uuid.UUID, amountMicro int64, intentID uuid.UUID, provider, providerRef string) ([]byte, error) {
	return json.Marshal(settleBalanceOutboxPayload{
		CustomerID:           customerID.String(),
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: ledgerIdempotencyKey(intentID),
		PaymentIntentID:      intentID.String(),
		Provider:             provider,
		ProviderRef:          providerRef,
	})
}

func marshalApplyChargebackOutbox(intentID, customerID uuid.UUID, amountMicro int64, providerDisputeID string) ([]byte, error) {
	return json.Marshal(applyChargebackPayload(intentID, customerID, amountMicro, providerDisputeID))
}

func marshalReverseChargebackOutbox(intentID, customerID uuid.UUID, amountMicro int64, providerDisputeID string) ([]byte, error) {
	return json.Marshal(reverseChargebackPayload(intentID, customerID, amountMicro, providerDisputeID))
}

func marshalReverseBalanceOutbox(intentID, customerID uuid.UUID, amountMicro int64, providerRefundID string) ([]byte, error) {
	return json.Marshal(reverseBalancePayload(intentID, customerID, amountMicro, providerRefundID))
}
