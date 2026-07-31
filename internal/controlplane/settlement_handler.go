package controlplane

import (
	"context"
	"errors"
	"espx/internal/domain"
	db "espx/internal/domain/db"

	"github.com/google/uuid"
)

func (h *SettlementHandler) PaymentSettlement() domain.PaymentSettlement {
	return handlerPaymentSettlement{h: h}
}

type handlerPaymentSettlement struct {
	h *SettlementHandler
}

func (a handlerPaymentSettlement) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentCredit(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
	return applied, ledgerEntryID, mapSettlementDomainError(err)
}

func (a handlerPaymentSettlement) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
	return applied, ledgerEntryID, mapSettlementDomainError(err)
}

func (a handlerPaymentSettlement) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
	return applied, ledgerEntryID, mapSettlementDomainError(err)
}

func (a handlerPaymentSettlement) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
	return applied, ledgerEntryID, mapSettlementDomainError(err)
}

func (a handlerPaymentSettlement) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, error) {
	entry, _, err := a.h.ledgerEntry(ctx, paymentIntentID)
	return entry, err
}

type settlementCreditParams struct {
	CustomerID           uuid.UUID
	AmountMicro          int64
	LedgerIdempotencyKey string
	PaymentIntentID      uuid.UUID
	Provider             string
	ProviderRef          string
}

type settlementRefundParams struct {
	CustomerID           uuid.UUID
	AmountMicro          int64
	LedgerIdempotencyKey string
	PaymentIntentID      uuid.UUID
	Provider             string
	ProviderRefundID     string
}

type settlementChargebackParams struct {
	CustomerID           uuid.UUID
	AmountMicro          int64
	LedgerIdempotencyKey string
	PaymentIntentID      uuid.UUID
	Provider             string
	ProviderDisputeID    string
}

type settlementBatchParams struct {
	Credits             []settlementCreditParams
	Refunds             []settlementRefundParams
	Chargebacks         []settlementChargebackParams
	ChargebackReversals []settlementChargebackParams
}

type settlementBatchItemResult struct {
	Applied       bool
	LedgerEntryID int64
	Err           error
}

type settlementBatchResult struct {
	CreditResults             []settlementBatchItemResult
	RefundResults             []settlementBatchItemResult
	ChargebackResults         []settlementBatchItemResult
	ChargebackReversalResults []settlementBatchItemResult
}

func (h *SettlementHandler) applyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	return h.service.ApplyPaymentCredit(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
}

func (h *SettlementHandler) applyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	return h.service.ApplyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
}

func (h *SettlementHandler) applyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return h.service.ApplyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (h *SettlementHandler) applyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return h.service.ApplyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (h *SettlementHandler) ledgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, db.BalanceLedger, error) {
	found, entry, refundTotal, chargebackTotal, reversalTotal, err := h.service.GetLedgerEntry(ctx, paymentIntentID)
	if err != nil {
		return domain.PaymentLedgerEntry{}, db.BalanceLedger{}, err
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
	return out, entry, nil
}

func (h *SettlementHandler) blockIP(ctx context.Context, ip, source string) error {
	return h.service.BlockIP(ctx, ip, source)
}

func (h *SettlementHandler) enqueueFraudThreat(ctx context.Context, payload FraudThreatPayload) error {
	return h.service.EnqueueFraudThreat(ctx, payload)
}

func (h *SettlementHandler) applyCTVSettlement(ctx context.Context, settlementID string, customerID, campaignID uuid.UUID, spendMicro int64) (domain.CTVSettlementResult, error) {
	return h.service.ApplyCTVSettlement(ctx, settlementID, customerID, campaignID, spendMicro)
}

const batchSettlementMaxItems = 500

func (h *SettlementHandler) batchApplySettlement(ctx context.Context, batch settlementBatchParams) settlementBatchResult {
	var out settlementBatchResult
	for _, item := range batch.Credits {
		applied, ledgerEntryID, err := h.applyPaymentCredit(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderRef)
		out.CreditResults = append(out.CreditResults, settlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	for _, item := range batch.Refunds {
		applied, ledgerEntryID, err := h.applyPaymentRefund(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderRefundID)
		out.RefundResults = append(out.RefundResults, settlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	for _, item := range batch.Chargebacks {
		applied, ledgerEntryID, err := h.applyPaymentChargeback(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderDisputeID)
		out.ChargebackResults = append(out.ChargebackResults, settlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	for _, item := range batch.ChargebackReversals {
		applied, ledgerEntryID, err := h.applyPaymentChargebackReversal(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderDisputeID)
		out.ChargebackReversalResults = append(out.ChargebackReversalResults, settlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	return out
}

func mapSettlementDomainError(err error) error {
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
