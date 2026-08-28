package payment

import (
	"context"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type SettlementHost interface {
	ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error)
	ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error)
	ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error)
	ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error)
	GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (found bool, entry db.BalanceLedger, refundTotal, chargebackTotal, reversalTotal int64, err error)
	GetLedgerEntries(ctx context.Context, paymentIntentIDs []uuid.UUID) (map[uuid.UUID]domain.PaymentLedgerEntry, error)
	ApplyCTVSettlement(ctx context.Context, settlementID string, customerID, campaignID uuid.UUID, spendMicro int64) (domain.CTVSettlementResult, error)
	ErrCustomerNotFound() error
	ErrPaymentTopupNotFound() error
}
