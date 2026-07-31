package payment

import (
	"context"
	"fmt"
	"sync"

	"espx/internal/config"
	"espx/internal/controlplane/pb"
	"espx/internal/domain"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SettlementLedgerClient struct {
	cfg  *config.Config
	mu   sync.Mutex
	conn *grpc.ClientConn
	api  domain.PaymentSettlement
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
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.api = nil
		return err
	}
	return nil
}

func (c *SettlementLedgerClient) ensureClient() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api != nil {
		return nil
	}
	target := c.cfg.SettlementServerHost + ":" + c.cfg.SettlementServerPort
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial settlement %s: %w", target, err)
	}
	c.conn = conn
	c.api = newGRPCSettlementClient(pb.NewSettlementServiceClient(conn), string(c.cfg.SettlementInternalToken))
	return nil
}

func (c *SettlementLedgerClient) getAPI() domain.PaymentSettlement {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api
}
