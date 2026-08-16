package payment

import (
	"fmt"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/config"
)

const (
	CryptoProviderGeneric   = "crypto"
	CryptoProviderBTCPay    = "btcpay"
	CryptoProviderCryptomus = "cryptomus"
)

func NormalizeCryptoProvider(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case CryptoProviderBTCPay, "btcpay-server":
		return CryptoProviderBTCPay
	case CryptoProviderCryptomus:
		return CryptoProviderCryptomus
	case CryptoProviderGeneric, "":
		return CryptoProviderGeneric
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

func CreateCryptoCheckout(
	cfg *config.Config,
	provider string,
	amountMicro int64,
	idempotencyKey string,
) (providerRef, checkoutURL string, err error) {
	minPayment := int64(0)
	if cfg != nil {
		minPayment = cfg.CryptoMinPaymentMicro
	}
	if minPayment > 0 && amountMicro < minPayment {
		return "", "", fmt.Errorf("amount %d micro is below minimum payment %d micro", amountMicro, minPayment)
	}
	prov := NormalizeCryptoProvider(provider)
	ref := prov + "_" + idempotencyKey
	switch prov {
	case CryptoProviderBTCPay:
		return ref, "https://checkout.btcpay.dev/pay/" + idempotencyKey, nil
	case CryptoProviderCryptomus:
		return ref, "https://checkout.cryptomus.dev/pay/" + idempotencyKey, nil
	default:
		return "tx_crypto_" + idempotencyKey, "https://checkout.crypto.dev/pay/" + idempotencyKey, nil
	}
}
