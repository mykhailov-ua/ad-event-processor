package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrSettlementCustomerNotFound = errors.New("settlement customer not found")
	ErrSettlementTopupNotFound    = errors.New("settlement payment topup not found")
)

type PaymentLedgerEntry struct {
	Found                        bool
	HasTopup                     bool
	TopupAmountMicro             int64
	RefundTotalMicro             int64
	ChargebackTotalMicro         int64
	ChargebackReversalTotalMicro int64
}

type PaymentSettlement interface {
	ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (applied bool, ledgerEntryID int64, err error)
	ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (applied bool, ledgerEntryID int64, err error)
	ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (applied bool, ledgerEntryID int64, err error)
	ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (applied bool, ledgerEntryID int64, err error)
	GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (PaymentLedgerEntry, error)
}

func IsSettlementNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrSettlementCustomerNotFound) || errors.Is(err, ErrSettlementTopupNotFound)
}
