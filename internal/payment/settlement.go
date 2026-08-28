package payment

import (
	"context"
	"errors"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type SettlementHandler struct {
	host SettlementHost
	cfg  *config.Config
}

func NewSettlementHandler(host SettlementHost, cfg *config.Config) *SettlementHandler {
	return &SettlementHandler{
		host: host,
		cfg:  cfg,
	}
}

func (h *SettlementHandler) PaymentSettlement() domain.PaymentSettlement {
	return handlerPaymentSettlement{h: h}
}

type handlerPaymentSettlement struct {
	h *SettlementHandler
}

func (a handlerPaymentSettlement) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentCredit(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
	return applied, ledgerEntryID, mapSettlementDomainError(a.h.host, err)
}

func (a handlerPaymentSettlement) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
	return applied, ledgerEntryID, mapSettlementDomainError(a.h.host, err)
}

func (a handlerPaymentSettlement) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
	return applied, ledgerEntryID, mapSettlementDomainError(a.h.host, err)
}

func (a handlerPaymentSettlement) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	applied, ledgerEntryID, err := a.h.applyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
	return applied, ledgerEntryID, mapSettlementDomainError(a.h.host, err)
}

func (a handlerPaymentSettlement) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, error) {
	entry, _, err := a.h.ledgerEntry(ctx, paymentIntentID)
	return entry, err
}

func (a handlerPaymentSettlement) GetLedgerEntries(ctx context.Context, paymentIntentIDs []uuid.UUID) (map[uuid.UUID]domain.PaymentLedgerEntry, error) {
	return a.h.host.GetLedgerEntries(ctx, paymentIntentIDs)
}

type SettlementCreditParams struct {
	CustomerID           uuid.UUID
	AmountMicro          int64
	LedgerIdempotencyKey string
	PaymentIntentID      uuid.UUID
	Provider             string
	ProviderRef          string
}

type SettlementRefundParams struct {
	CustomerID           uuid.UUID
	AmountMicro          int64
	LedgerIdempotencyKey string
	PaymentIntentID      uuid.UUID
	Provider             string
	ProviderRefundID     string
}

type SettlementChargebackParams struct {
	CustomerID           uuid.UUID
	AmountMicro          int64
	LedgerIdempotencyKey string
	PaymentIntentID      uuid.UUID
	Provider             string
	ProviderDisputeID    string
}

type SettlementBatchParams struct {
	Credits             []SettlementCreditParams
	Refunds             []SettlementRefundParams
	Chargebacks         []SettlementChargebackParams
	ChargebackReversals []SettlementChargebackParams
}

type SettlementBatchItemResult struct {
	Applied       bool
	LedgerEntryID int64
	Err           error
}

type SettlementBatchResult struct {
	CreditResults             []SettlementBatchItemResult
	RefundResults             []SettlementBatchItemResult
	ChargebackResults         []SettlementBatchItemResult
	ChargebackReversalResults []SettlementBatchItemResult
}

func (h *SettlementHandler) applyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	return h.host.ApplyPaymentCredit(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
}

func (h *SettlementHandler) applyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	return h.host.ApplyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
}

func (h *SettlementHandler) applyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return h.host.ApplyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (h *SettlementHandler) applyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return h.host.ApplyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (h *SettlementHandler) ledgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, db.BalanceLedger, error) {
	found, entry, refundTotal, chargebackTotal, reversalTotal, err := h.host.GetLedgerEntry(ctx, paymentIntentID)
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

func (h *SettlementHandler) ApplyCTVSettlement(ctx context.Context, settlementID string, customerID, campaignID uuid.UUID, spendMicro int64) (domain.CTVSettlementResult, error) {
	return h.host.ApplyCTVSettlement(ctx, settlementID, customerID, campaignID, spendMicro)
}

func (h *SettlementHandler) BatchApplySettlement(ctx context.Context, batch SettlementBatchParams) SettlementBatchResult {
	var out SettlementBatchResult
	for _, item := range batch.Credits {
		applied, ledgerEntryID, err := h.applyPaymentCredit(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderRef)
		out.CreditResults = append(out.CreditResults, SettlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	for _, item := range batch.Refunds {
		applied, ledgerEntryID, err := h.applyPaymentRefund(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderRefundID)
		out.RefundResults = append(out.RefundResults, SettlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	for _, item := range batch.Chargebacks {
		applied, ledgerEntryID, err := h.applyPaymentChargeback(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderDisputeID)
		out.ChargebackResults = append(out.ChargebackResults, SettlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	for _, item := range batch.ChargebackReversals {
		applied, ledgerEntryID, err := h.applyPaymentChargebackReversal(ctx, item.CustomerID, item.AmountMicro, item.LedgerIdempotencyKey, item.PaymentIntentID, item.Provider, item.ProviderDisputeID)
		out.ChargebackReversalResults = append(out.ChargebackReversalResults, SettlementBatchItemResult{Applied: applied, LedgerEntryID: ledgerEntryID, Err: err})
	}
	return out
}

func mapSettlementDomainError(host SettlementHost, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, host.ErrCustomerNotFound()) {
		return domain.ErrSettlementCustomerNotFound
	}
	if errors.Is(err, host.ErrPaymentTopupNotFound()) {
		return domain.ErrSettlementTopupNotFound
	}
	return err
}
