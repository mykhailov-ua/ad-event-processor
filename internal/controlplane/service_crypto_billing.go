package controlplane

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/payment"
)

func (s *Service) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	if s == nil {
		return errValidation("service unavailable")
	}
	pool := s.paymentPool
	if pool == nil {
		pool = s.pool
	}
	if pool == nil {
		return errValidation("database unavailable")
	}
	ps := payment.NewService(pool, s.cfg)
	return ps.ProcessCryptoWebhook(ctx, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}
