package payment

import (
	"context"
	"fmt"
	"sync"

	"espx/internal/config"
	"espx/internal/domain"

	"github.com/google/uuid"
)

type SettlementLedgerClient struct {
	cfg     *config.Config
	mu      sync.Mutex
	closeFn func()
	api     domain.PaymentSettlement
}

func NewSettlementLedgerClient(cfg *config.Config) *SettlementLedgerClient {
	return &SettlementLedgerClient{cfg: cfg}
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
	if err := c.ensureClient(); err != nil {
		return PaymentIntentLedger{}, err
	}
	entry, err := c.getAPI().GetLedgerEntry(ctx, intentID)
	if err != nil {
		return PaymentIntentLedger{}, fmt.Errorf("settlement GetLedgerEntry: %w", err)
	}
	out := PaymentIntentLedger{
		RefundMicro:             entry.RefundTotalMicro,
		ChargebackMicro:         entry.ChargebackTotalMicro,
		ChargebackReversalMicro: entry.ChargebackReversalTotalMicro,
	}
	if entry.Found && entry.HasTopup {
		out.HasTopup = true
		out.TopupMicro = entry.TopupAmountMicro
	}
	return out, nil
}

func (c *SettlementLedgerClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closeFn != nil {
		c.closeFn()
		c.closeFn = nil
	}
	c.api = nil
	return nil
}

func (c *SettlementLedgerClient) ensureClient() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api != nil {
		return nil
	}
	if c.cfg != nil && !c.cfg.SettlementGRPCEnabled {
		return fmt.Errorf("settlement API not injected")
	}
	api, closeFn, err := OpenSettlementAPIOrDial(context.Background(), c.cfg)
	if err != nil {
		return err
	}
	if api == nil {
		return fmt.Errorf("settlement API not injected")
	}
	c.api = api
	c.closeFn = closeFn
	return nil
}

func (c *SettlementLedgerClient) getAPI() domain.PaymentSettlement {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api
}
