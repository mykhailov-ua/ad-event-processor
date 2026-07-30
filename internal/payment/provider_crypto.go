package payment

import (
	"context"
	"fmt"
)

type CryptoProvider struct {
	confirmationDepth int
	minPaymentMicro   int64
	webhookSecret     string
}

func NewCryptoProvider(confirmationDepth int, minPaymentMicro int64, webhookSecret string) *CryptoProvider {
	if confirmationDepth <= 0 {
		confirmationDepth = 12
	}
	return &CryptoProvider{
		confirmationDepth: confirmationDepth,
		minPaymentMicro:   minPaymentMicro,
		webhookSecret:     webhookSecret,
	}
}

func (p *CryptoProvider) Name() string {
	return "crypto"
}

func (p *CryptoProvider) CreateCheckout(ctx context.Context, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (string, string, error) {
	if amountMicro < p.minPaymentMicro {
		return "", "", fmt.Errorf("amount %d micro is below minimum payment %d micro", amountMicro, p.minPaymentMicro)
	}
	_ = ctx
	providerRef := "tx_crypto_" + idempotencyKey
	checkoutURL := "https://checkout.crypto.dev/pay/" + idempotencyKey
	return providerRef, checkoutURL, nil
}
