package billingadmin

import (
	"context"
	"errors"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/payment"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CryptoBillingHost interface {
	PaymentPool() *pgxpool.Pool
	FallbackPool() *pgxpool.Pool
	Config() *config.Config
	ErrValidation(msg string) error
}

type CryptoBilling struct {
	host CryptoBillingHost
}

func NewCryptoBilling(host CryptoBillingHost) *CryptoBilling {
	return &CryptoBilling{host: host}
}

func ProcessCryptoWebhook(ctx context.Context, host CryptoBillingHost, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	return NewCryptoBilling(host).ProcessCryptoWebhook(ctx, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}

func (c *CryptoBilling) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	if c == nil || c.host == nil {
		return errors.New("service unavailable")
	}
	pool := c.host.PaymentPool()
	if pool == nil {
		pool = c.host.FallbackPool()
	}
	if pool == nil {
		return c.host.ErrValidation("database unavailable")
	}
	ps := payment.NewService(pool, c.host.Config())
	return ps.ProcessCryptoWebhook(ctx, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}
