package payment

import "github.com/google/uuid"

func refundLedgerIdempotencyKey(providerRefundID string) string {
	return "refund:" + providerRefundID
}

const OutboxEventReverseBalance = "REVERSE_BALANCE"

type ReverseBalancePayload struct {
	CustomerID           string `json:"customer_id"`
	AmountMicro          int64  `json:"amount_micro"`
	LedgerIdempotencyKey string `json:"ledger_idempotency_key"`
	PaymentIntentID      string `json:"payment_intent_id"`
	Provider             string `json:"provider"`
	ProviderRefundID     string `json:"provider_refund_id"`
}

func reverseBalancePayload(intentID, customerID uuid.UUID, amountMicro int64, providerRefundID string) ReverseBalancePayload {
	return ReverseBalancePayload{
		CustomerID:           customerID.String(),
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: refundLedgerIdempotencyKey(providerRefundID),
		PaymentIntentID:      intentID.String(),
		Provider:             "stripe",
		ProviderRefundID:     providerRefundID,
	}
}
