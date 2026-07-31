package controlplane

import (
	"context"
	"errors"

	"espx/internal/domain"

	"github.com/google/uuid"
)

type settlementBridge struct {
	svc *Service
}

func (b settlementBridge) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	applied, ledgerEntryID, err := b.svc.ApplyPaymentCredit(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
	return applied, ledgerEntryID, mapSettlementBridgeError(err)
}

func (b settlementBridge) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	applied, ledgerEntryID, err := b.svc.ApplyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
	return applied, ledgerEntryID, mapSettlementBridgeError(err)
}

func (b settlementBridge) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	applied, ledgerEntryID, err := b.svc.ApplyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
	return applied, ledgerEntryID, mapSettlementBridgeError(err)
}

func (b settlementBridge) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	applied, ledgerEntryID, err := b.svc.ApplyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
	return applied, ledgerEntryID, mapSettlementBridgeError(err)
}

func (b settlementBridge) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, error) {
	found, entry, refundTotal, chargebackTotal, reversalTotal, err := b.svc.GetLedgerEntry(ctx, paymentIntentID)
	if err != nil {
		return domain.PaymentLedgerEntry{}, err
	}
	out := domain.PaymentLedgerEntry{
		Found:                        found,
		RefundTotalMicro:             refundTotal,
		ChargebackTotalMicro:         chargebackTotal,
		ChargebackReversalTotalMicro: reversalTotal,
	}
	if found {
		out.HasTopup = true
		out.TopupAmountMicro = entry.Amount
	}
	return out, nil
}

func mapSettlementBridgeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrCustomerNotFound) {
		return domain.ErrSettlementCustomerNotFound
	}
	if errors.Is(err, ErrPaymentTopupNotFound) {
		return domain.ErrSettlementTopupNotFound
	}
	return err
}
