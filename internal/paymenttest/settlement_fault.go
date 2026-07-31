package paymenttest

import (
	"context"
	"fmt"
	"sync"

	"espx/internal/domain"

	"github.com/google/uuid"
)

type SettlementFaultGate struct {
	api  domain.PaymentSettlement
	mu   sync.Mutex
	down bool
}

func NewSettlementFaultGate(api domain.PaymentSettlement) *SettlementFaultGate {
	return &SettlementFaultGate{api: api}
}

func (g *SettlementFaultGate) SetDown(down bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.down = down
}

func (g *SettlementFaultGate) downErr() error {
	return fmt.Errorf("connection refused")
}

func (g *SettlementFaultGate) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	g.mu.Lock()
	down := g.down
	g.mu.Unlock()
	if down {
		return false, 0, g.downErr()
	}
	return g.api.ApplyPaymentCredit(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRef)
}

func (g *SettlementFaultGate) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	g.mu.Lock()
	down := g.down
	g.mu.Unlock()
	if down {
		return false, 0, g.downErr()
	}
	return g.api.ApplyPaymentRefund(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerRefundID)
}

func (g *SettlementFaultGate) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	g.mu.Lock()
	down := g.down
	g.mu.Unlock()
	if down {
		return false, 0, g.downErr()
	}
	return g.api.ApplyPaymentChargeback(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (g *SettlementFaultGate) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	g.mu.Lock()
	down := g.down
	g.mu.Unlock()
	if down {
		return false, 0, g.downErr()
	}
	return g.api.ApplyPaymentChargebackReversal(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID)
}

func (g *SettlementFaultGate) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, error) {
	g.mu.Lock()
	down := g.down
	g.mu.Unlock()
	if down {
		return domain.PaymentLedgerEntry{}, g.downErr()
	}
	return g.api.GetLedgerEntry(ctx, paymentIntentID)
}
