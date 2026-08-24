package payment

import (
	"context"
	"fmt"
	"sync"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type SettlementLedgerClient struct {
	mu  sync.Mutex
	api domain.PaymentSettlement
}

func NewSettlementLedgerClient(cfg *config.Config) *SettlementLedgerClient {
	return &SettlementLedgerClient{}
}

func (c *SettlementLedgerClient) SetSettlementAPI(api domain.PaymentSettlement) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.api = api
}

type PaymentIntentLedger struct {
	TopupMicro              int64
	RefundMicro             int64
	ChargebackMicro         int64
	ChargebackReversalMicro int64
	HasTopup                bool
}

func (c *SettlementLedgerClient) GetPaymentIntentLedger(ctx context.Context, intentID uuid.UUID) (PaymentIntentLedger, error) {
	ledgers, err := c.GetPaymentIntentLedgers(ctx, []uuid.UUID{intentID})
	if err != nil {
		return PaymentIntentLedger{}, err
	}
	return ledgers[intentID], nil
}

func (c *SettlementLedgerClient) GetPaymentIntentLedgers(ctx context.Context, intentIDs []uuid.UUID) (map[uuid.UUID]PaymentIntentLedger, error) {
	out := make(map[uuid.UUID]PaymentIntentLedger, len(intentIDs))
	if len(intentIDs) == 0 {
		return out, nil
	}
	if err := c.ensureClient(); err != nil {
		return nil, err
	}
	entries, err := c.getAPI().GetLedgerEntries(ctx, intentIDs)
	if err != nil {
		return nil, fmt.Errorf("settlement GetLedgerEntries: %w", err)
	}
	for id, entry := range entries {
		out[id] = paymentIntentLedgerFromEntry(entry)
	}
	return out, nil
}

func paymentIntentLedgerFromEntry(entry domain.PaymentLedgerEntry) PaymentIntentLedger {
	out := PaymentIntentLedger{
		RefundMicro:             entry.RefundTotalMicro,
		ChargebackMicro:         entry.ChargebackTotalMicro,
		ChargebackReversalMicro: entry.ChargebackReversalTotalMicro,
	}
	if entry.HasTopup {
		out.HasTopup = true
		out.TopupMicro = entry.TopupAmountMicro
	}
	return out
}

func (c *SettlementLedgerClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.api = nil
	return nil
}

func (c *SettlementLedgerClient) ensureClient() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api != nil {
		return nil
	}
	return fmt.Errorf("settlement API not injected")
}

func (c *SettlementLedgerClient) getAPI() domain.PaymentSettlement {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api
}
