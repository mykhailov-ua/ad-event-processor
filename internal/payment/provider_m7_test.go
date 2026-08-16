package payment_test

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/payment"

	"github.com/stretchr/testify/require"
)

func TestVerifyBTCPayWebhookSignature(t *testing.T) {
	body := []byte(`{"id":"evt1","provider_ref":"tx_1"}`)
	secret := "btcpay-secret"
	mac := payment.VerifyBTCPayWebhookSignature(body, "bad", secret)
	require.False(t, mac)
}

func TestVerifyCryptomusWebhookSignature_rejectsMissingSign(t *testing.T) {
	body := []byte(`{"order_id":"1","amount":"10"}`)
	require.False(t, payment.VerifyCryptomusWebhookSignature(body, "", "api-key"))
}

func TestCreateCryptoCheckout_providers(t *testing.T) {
	ref, url, err := payment.CreateCryptoCheckout(nil, payment.CryptoProviderBTCPay, 50_000_000, "idem-1")
	require.NoError(t, err)
	require.Contains(t, ref, "btcpay_")
	require.Contains(t, url, "btcpay")

	ref, url, err = payment.CreateCryptoCheckout(nil, payment.CryptoProviderCryptomus, 50_000_000, "idem-2")
	require.NoError(t, err)
	require.Contains(t, ref, "cryptomus_")
	require.Contains(t, url, "cryptomus")
}
