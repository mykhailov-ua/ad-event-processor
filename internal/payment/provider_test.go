package payment

import (
	"testing"

	"espx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider_mockByDefault(t *testing.T) {
	p := NewProvider(&config.Config{})
	_, ok := p.(*MockProvider)
	assert.True(t, ok)
}

func TestNewProvider_stripeWhenKeySet(t *testing.T) {
	p := NewProvider(&config.Config{StripeSecretKey: "sk_test_x"})
	_, ok := p.(*StripeProvider)
	assert.True(t, ok)
}

func TestStripeProvider_CreateCheckout_unalignedAmount(t *testing.T) {
	p := NewStripeProvider(&config.Config{StripeSecretKey: "sk_test_x"})
	_, _, err := p.CreateCheckout(t.Context(), 10_001, "USD", nil, "idem-1")
	require.Error(t, err)
}

func TestStripeProvider_CreateCheckout_notWired(t *testing.T) {
	p := NewStripeProvider(&config.Config{StripeSecretKey: "sk_test_x"})
	_, _, err := p.CreateCheckout(t.Context(), 10_000_000, "USD", nil, "idem-1")
	require.ErrorIs(t, err, ErrProviderNotConfigured)
}

func TestCryptoProvider_CreateCheckout_belowMinimum(t *testing.T) {
	p := NewCryptoProvider(12, 10_000_000, "whsec_test")
	_, _, err := p.CreateCheckout(t.Context(), 5_000_000, "USDT", nil, "idem-crypto-1")
	require.Error(t, err)
}

func TestCryptoProvider_CreateCheckout_ok(t *testing.T) {
	p := NewCryptoProvider(12, 10_000_000, "whsec_test")
	ref, url, err := p.CreateCheckout(t.Context(), 50_000_000, "USDT", nil, "idem-crypto-2")
	require.NoError(t, err)
	assert.Equal(t, "crypto", p.Name())
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
