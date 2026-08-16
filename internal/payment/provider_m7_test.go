package payment_test

import (
	"encoding/json"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/payment"

	"github.com/stretchr/testify/require"
)

func TestCryptoWebhook_Signature_BTCPay(t *testing.T) {
	body := []byte(`{"deliveryId":"del-1","webhookId":"wh-1","invoiceId":"inv-1","type":"InvoiceSettled"}`)
	secret := "btcpay-store-secret"

	sig := payment.SignBTCPayWebhookBody(body, secret)
	require.True(t, payment.VerifyBTCPayWebhookSignature(body, sig, secret))
	require.True(t, payment.VerifyBTCPayWebhookSignature(body, sig[7:], secret))
	require.False(t, payment.VerifyBTCPayWebhookSignature(body, "sha256=deadbeef", secret))
	require.False(t, payment.VerifyBTCPayWebhookSignature(append(body, ' '), sig, secret))
}

func TestCryptoWebhook_Signature_Cryptomus(t *testing.T) {
	apiKey := "cryptomus_api_key_test"
	fields := map[string]any{
		"order_id": "ord_gm_m5",
		"amount":   "100.00",
		"currency": "USDT",
		"status":   "paid",
	}
	body, sign, err := payment.SignCryptomusWebhookFields(fields, apiKey)
	require.NoError(t, err)
	require.True(t, payment.VerifyCryptomusWebhookSignature(body, sign, apiKey))

	tampered := append([]byte(nil), body...)
	tampered[len(tampered)-2] ^= 0xff
	require.False(t, payment.VerifyCryptomusWebhookSignature(tampered, sign, apiKey))
}

func TestCryptoWebhook_Signature_Cryptomus_vector(t *testing.T) {
	apiKey := "test_api_key"
	raw := []byte(`{"uuid":"evt-1","order_id":"1","amount":"10.00","currency":"USDT"}`)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(raw, &fields))
	body, sign, err := payment.SignCryptomusWebhookFields(fields, apiKey)
	require.NoError(t, err)
	require.True(t, payment.VerifyCryptomusWebhookSignature(body, sign, apiKey))
	require.False(t, payment.VerifyCryptomusWebhookSignature(body, "", apiKey))
}

func TestVerifyCryptomusWebhookSignature_rejectsMissingSign(t *testing.T) {
	body := []byte(`{"order_id":"1","amount":"10"}`)
	require.False(t, payment.VerifyCryptomusWebhookSignature(body, "", "api-key"))
}

func TestCreateCryptoCheckout_providers(t *testing.T) {
	res, err := payment.CreateCryptoCheckout(nil, payment.CryptoProviderBTCPay, 50_000_000, "idem-1")
	require.NoError(t, err)
	require.Contains(t, res.ProviderRef, "btcpay_")
	require.Contains(t, res.CheckoutURL, "btcpay")
	require.NotEmpty(t, res.DepositAddress)
	require.Equal(t, "USDT TRC-20", res.DepositNetwork)

	res, err = payment.CreateCryptoCheckout(nil, payment.CryptoProviderCryptomus, 50_000_000, "idem-2")
	require.NoError(t, err)
	require.Contains(t, res.ProviderRef, "cryptomus_")
	require.Contains(t, res.CheckoutURL, "cryptomus")
	require.NotEmpty(t, res.DepositAddress)
}
