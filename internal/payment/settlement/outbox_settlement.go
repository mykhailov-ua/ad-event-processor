package settlement

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/payment/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

type settlementPayload interface {
	customerID() string
	paymentIntentID() string
}

func (p SettleBalancePayload) customerID() string          { return p.CustomerID }
func (p SettleBalancePayload) paymentIntentID() string     { return p.PaymentIntentID }
func (p ReverseBalancePayload) customerID() string         { return p.CustomerID }
func (p ReverseBalancePayload) paymentIntentID() string    { return p.PaymentIntentID }
func (p ApplyChargebackPayload) customerID() string        { return p.CustomerID }
func (p ApplyChargebackPayload) paymentIntentID() string   { return p.PaymentIntentID }
func (p ReverseChargebackPayload) customerID() string      { return p.CustomerID }
func (p ReverseChargebackPayload) paymentIntentID() string { return p.PaymentIntentID }

func decodeOutboxPayload[T any](outboxEvent db.PaymentPaymentOutbox, label string) (T, error) {
	payload, err := coldpath.UnmarshalStrict[T](outboxEvent.Payload)
	if err != nil {
		return payload, fmt.Errorf("failed to unmarshal %s payload: %w", label, err)
	}
	return payload, nil
}

func (w *OutboxWorker) requireSettlementAPI() (domain.PaymentSettlement, error) {
	api := w.getSettlementAPI()
	if api == nil {
		return nil, fmt.Errorf("settlement client not connected")
	}
	return api, nil
}

func parseSettlementCustomerAndIntent(customerIDStr, paymentIntentIDStr string) (customerID, paymentIntentID uuid.UUID, err error) {
	customerID, err = uuid.Parse(customerIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid customer id: %w", err)
	}
	paymentIntentID, err = uuid.Parse(paymentIntentIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid payment intent id: %w", err)
	}
	return customerID, paymentIntentID, nil
}

func paymentIntentIDFromPayload[T settlementPayload](payload []byte) (string, bool) {
	p, err := coldpath.UnmarshalStrict[T](payload)
	if err != nil {
		return "", false
	}
	if p.paymentIntentID() == "" {
		return "", false
	}
	return p.paymentIntentID(), true
}

func applySettlementOutbox[T settlementPayload](
	ctx context.Context,
	outboxWorker *OutboxWorker,
	outboxEvent db.PaymentPaymentOutbox,
	label string,
	apply func(api domain.PaymentSettlement, customerID, paymentIntentID uuid.UUID, payload T) error,
) error {
	payload, err := decodeOutboxPayload[T](outboxEvent, label)
	if err != nil {
		return err
	}
	api, err := outboxWorker.requireSettlementAPI()
	if err != nil {
		return err
	}
	customerID, paymentIntentID, err := parseSettlementCustomerAndIntent(payload.customerID(), payload.paymentIntentID())
	if err != nil {
		return err
	}
	return apply(api, customerID, paymentIntentID, payload)
}
