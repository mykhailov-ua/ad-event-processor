package payment

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"ad-event-processor/internal/config"
)

const (
	CryptoProviderGeneric   = "crypto"
	CryptoProviderBTCPay    = "btcpay"
	CryptoProviderCryptomus = "cryptomus"

	cryptoDepositNetwork = "USDT TRC-20"
)

type CryptoCheckoutResult struct {
	ProviderRef    string
	CheckoutURL    string
	DepositAddress string
	DepositNetwork string
	DepositQRSVG   string
}

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
) (CryptoCheckoutResult, error) {
	minPayment := int64(0)
	if cfg != nil {
		minPayment = cfg.CryptoMinPaymentMicro
	}
	if minPayment > 0 && amountMicro < minPayment {
		return CryptoCheckoutResult{}, fmt.Errorf("amount %d micro is below minimum payment %d micro", amountMicro, minPayment)
	}
	prov := NormalizeCryptoProvider(provider)
	ref := prov + "_" + idempotencyKey
	deposit := tronDepositAddress(ref)
	var checkoutURL string
	switch prov {
	case CryptoProviderBTCPay:
		checkoutURL = "https://checkout.btcpay.dev/pay/" + idempotencyKey
	case CryptoProviderCryptomus:
		checkoutURL = "https://checkout.cryptomus.dev/pay/" + idempotencyKey
	default:
		ref = "tx_crypto_" + idempotencyKey
		checkoutURL = "https://checkout.crypto.dev/pay/" + idempotencyKey
	}
	return CryptoCheckoutResult{
		ProviderRef:    ref,
		CheckoutURL:    checkoutURL,
		DepositAddress: deposit,
		DepositNetwork: cryptoDepositNetwork,
		DepositQRSVG:   cryptoDepositQRSVG(deposit),
	}, nil
}

const tronBase58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func tronDepositAddress(seed string) string {
	sum := sha256.Sum256([]byte("usdt-trc20:" + seed))
	var out [34]byte
	out[0] = 'T'
	for i := 1; i < len(out); i++ {
		out[i] = tronBase58Alphabet[int(sum[(i-1)%len(sum)])%len(tronBase58Alphabet)]
	}
	return string(out[:])
}
