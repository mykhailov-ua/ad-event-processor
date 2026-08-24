package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ledger"

	"github.com/redis/go-redis/v9"
)

const invoiceRetryIdempotencyTTL = 24 * time.Hour

type invoiceDeliveryRetryer struct {
	ledger *ledger.Service
	redisShards   []redis.UniversalClient
}

func newInvoiceDeliveryRetryer(svc *ledger.Service, redisShards []redis.UniversalClient) *invoiceDeliveryRetryer {
	if svc == nil {
		return nil
	}
	return &invoiceDeliveryRetryer{ledger: svc, redisShards: redisShards}
}

func (r *invoiceDeliveryRetryer) RetryInvoiceDelivery(ctx context.Context, invoice *domain.Invoice, idempotencyKey string) error {
	if r == nil || r.ledger == nil {
		return fmt.Errorf("invoice delivery not configured")
	}
	if invoice == nil {
		return fmt.Errorf("invoice required")
	}
	if idempotencyKey != "" && len(r.redisShards) > 0 {
		redisClient := PickHealthyControlShard(r.redisShards)
		if redisClient != nil {
			idemKey := fmt.Sprintf("billing:invoice-retry:%s:%s", invoice.ID, idempotencyKey)
			ok, err := redisClient.SetNX(ctx, idemKey, "1", invoiceRetryIdempotencyTTL).Result()
			if err != nil {
				return fmt.Errorf("invoice retry idempotency: %w", err)
			}
			if !ok {
				return nil
			}
		}
	}
	return r.ledger.RetryInvoiceDelivery(ctx, invoice, idempotencyKey)
}
