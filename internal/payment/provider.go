package payment

import (
	"context"
	"log/slog"

	"espx/internal/config"
)

type ParsedEvent struct {
	EventID         string
	EventType       string
	PaymentIntentID string
	AmountMicro     int64
	ProviderRef     string
}

type PaymentProvider interface {
	Name() string
	CreateCheckout(ctx context.Context, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (providerRef string, checkoutURL string, err error)
}

type Provider = PaymentProvider

func StripeConfigured(cfg *config.Config) bool {
	return cfg != nil && string(cfg.StripeSecretKey) != ""
}

func CryptoConfigured(cfg *config.Config) bool {
	return cfg != nil && string(cfg.CryptoWebhookSecret) != ""
}

func NewProvider(cfg *config.Config) Provider {
	if StripeConfigured(cfg) {
		return NewStripeProvider(cfg)
	}
	return NewMockProvider()
}

func LogProviderMode(cfg *config.Config) {
	if StripeConfigured(cfg) {
		slog.Info("payment provider mode", "provider", "stripe", "checkout_api", "stripe_go")
		if string(cfg.StripeWebhookSecret) == "" {
			slog.Warn("STRIPE_WEBHOOK_SECRET unset; POST /webhooks/stripe returns 503")
		}
		if string(cfg.PaymentInternalToken) == "" {
			slog.Warn("PAYMENT_INTERNAL_TOKEN unset; gRPC CreatePaymentIntent rejects callers")
		}
	} else {
		slog.Info("payment provider mode", "provider", "mock")
	}
	logCryptoProviderMode(cfg)
}

func logCryptoProviderMode(cfg *config.Config) {
	if CryptoConfigured(cfg) {
		slog.Info("payment crypto webhook enabled", "provider", "crypto", "webhook_path", "/webhooks/crypto")
		return
	}
	slog.Warn("CRYPTO_WEBHOOK_SECRET unset; POST /webhooks/crypto returns 503")
}

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (mockProvider *MockProvider) Name() string {
	return "stripe"
}

func (mockProvider *MockProvider) CreateCheckout(ctx context.Context, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (string, string, error) {
	return "pi_mock_" + idempotencyKey, "https://checkout.stripe.dev/pay/mock_" + idempotencyKey, nil
}
