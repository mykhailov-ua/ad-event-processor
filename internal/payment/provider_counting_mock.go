package payment

import (
	"context"
	"sync/atomic"
)

type CountingMockProvider struct {
	calls atomic.Int32
}

func NewCountingMockProvider() *CountingMockProvider {
	return &CountingMockProvider{}
}

func (m *CountingMockProvider) Calls() int {
	return int(m.calls.Load())
}

func (m *CountingMockProvider) Name() string {
	return "stripe"
}

func (m *CountingMockProvider) CreateCheckout(ctx context.Context, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (string, string, error) {
	m.calls.Add(1)
	return "pi_mock_" + idempotencyKey, "https://checkout.stripe.dev/pay/mock_" + idempotencyKey, nil
}
