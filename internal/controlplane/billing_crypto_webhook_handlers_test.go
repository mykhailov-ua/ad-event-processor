package controlplane_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane"
	"github.com/bidshard/ad-event-processor/internal/payment"

	"github.com/stretchr/testify/require"
)

type stubCryptoProcessor struct {
	calls int
}

func (s *stubCryptoProcessor) ProcessCryptoWebhook(_ context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	s.calls++
	return nil
}

func TestBillingCryptoWebhook_BTCPaySignature(t *testing.T) {
	body := []byte(`{"id":"evt-btcpay-1","provider_ref":"btcpay_idem","amount_micro":50000000,"confirmations":12}`)
	secret := "btcpay-test-secret"
	sig := payment.SignBTCPayWebhookBody(body, secret)

	proc := &stubCryptoProcessor{}
	h := &controlplane.CryptoBillingWebhookHandlers{
		Processor:           proc,
		BTCPayWebhookSecret: secret,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/crypto/webhook", bytes.NewReader(body))
	req.Header.Set("X-Crypto-Provider", "btcpay")
	req.Header.Set("BTCPay-Sig", sig)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, 1, proc.calls)
}

func TestBillingCryptoWebhook_CryptomusSignature(t *testing.T) {
	apiKey := "cryptomus-test-key"
	fields := map[string]any{
		"id":            "evt-cm-1",
		"provider_ref":  "cryptomus_idem",
		"amount_micro":  int64(50_000_000),
		"confirmations": 12,
		"type":          "payment.succeeded",
	}
	body, sign, err := payment.SignCryptomusWebhookFields(fields, apiKey)
	require.NoError(t, err)

	proc := &stubCryptoProcessor{}
	h := &controlplane.CryptoBillingWebhookHandlers{
		Processor:       proc,
		CryptomusAPIKey: apiKey,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/crypto/webhook", bytes.NewReader(body))
	req.Header.Set("X-Crypto-Provider", "cryptomus")
	req.Header.Set("sign", sign)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, 1, proc.calls)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "ok", resp["status"])
}

func TestBillingCryptoWebhook_rejectsBadSignature(t *testing.T) {
	body := []byte(`{"id":"evt-bad","provider_ref":"ref"}`)
	h := &controlplane.CryptoBillingWebhookHandlers{
		Processor:           &stubCryptoProcessor{},
		BTCPayWebhookSecret: "secret",
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/crypto/webhook", bytes.NewReader(body))
	req.Header.Set("X-Crypto-Provider", "btcpay")
	req.Header.Set("BTCPay-Sig", "sha256=deadbeef")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
