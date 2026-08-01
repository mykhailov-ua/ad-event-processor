package payment

import (
	"testing"

	"espx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCheckout_mockByDefault(t *testing.T) {
	cfg := &config.Config{}
	ref, url, err := CreateCheckout(t.Context(), cfg, "stripe", 10_000_000, "USD", nil, "idem-1")
	require.NoError(t, err)
	assert.Equal(t, "pi_mock_idem-1", ref)
	assert.Contains(t, url, "idem-1")
}

func TestCreateCheckout_stripeWhenKeySet(t *testing.T) {
	cfg := &config.Config{StripeSecretKey: "sk_test_x"}
	_, _, err := CreateCheckout(t.Context(), cfg, "stripe", 10_000_000, "USD", nil, "idem-1")
	require.ErrorIs(t, err, ErrProviderNotConfigured)
}

func TestCreateCheckout_stripeUnalignedAmount(t *testing.T) {
	cfg := &config.Config{StripeSecretKey: "sk_test_x"}
	_, _, err := CreateCheckout(t.Context(), cfg, "stripe", 10_001, "USD", nil, "idem-1")
	require.Error(t, err)
}

func TestCreateCheckout_crypto_belowMinimum(t *testing.T) {
	cfg := &config.Config{CryptoMinPaymentMicro: 10_000_000}
	_, _, err := CreateCheckout(t.Context(), cfg, "crypto", 5_000_000, "USDT", nil, "idem-crypto-1")
	require.Error(t, err)
}

func TestCreateCheckout_crypto_ok(t *testing.T) {
	cfg := &config.Config{CryptoMinPaymentMicro: 10_000_000}
	ref, url, err := CreateCheckout(t.Context(), cfg, "crypto", 50_000_000, "USDT", nil, "idem-crypto-2")
	require.NoError(t, err)
	assert.Equal(t, "tx_crypto_idem-crypto-2", ref)
	assert.Contains(t, url, "idem-crypto-2")
}

func TestCryptoConfigured(t *testing.T) {
	assert.False(t, CryptoConfigured(&config.Config{}))
	assert.True(t, CryptoConfigured(&config.Config{CryptoWebhookSecret: "secret"}))
}

func TestMergeIntentMetadata_checkoutURL(t *testing.T) {
	raw, err := mergeIntentMetadata(map[string]string{"foo": "bar"}, "https://checkout.example/pay")
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"checkout_url":"https://checkout.example/pay"`)
	assert.Contains(t, string(raw), `"foo":"bar"`)
}
