package billingadmin

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"ad-event-processor/internal/payment"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type CryptoBillingWebhookProcessor interface {
	ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error
}

type CryptoWebhookHandlers struct {
	Processor           CryptoBillingWebhookProcessor
	CryptoWebhookSecret string
	BTCPayWebhookSecret string
	CryptomusAPIKey     string
}

type cryptoBillingEvent struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	TxHash        string `json:"tx_hash"`
	AmountMicro   int64  `json:"amount_micro"`
	Currency      string `json:"currency"`
	Confirmations int    `json:"confirmations"`
	ProviderRef   string `json:"provider_ref"`
}

func (h *CryptoWebhookHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Processor == nil {
		return
	}
	mux.HandleFunc("POST /api/v1/billing/crypto/webhook", h.handleCryptoWebhook)
}

func (h *CryptoWebhookHandlers) handleCryptoWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.PaymentWebhookMaxBody)
	if err != nil {
		return
	}
	if !h.verifySignature(body, r) {
		slog.Warn("invalid billing crypto webhook signature")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	event, err := coldpath.DecodeBody[cryptoBillingEvent](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if event.ID == "" || event.ProviderRef == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing event id or provider_ref")
		return
	}
	eventType := event.Type
	if eventType == "" {
		eventType = "payment.succeeded"
	}
	if err := h.Processor.ProcessCryptoWebhook(
		r.Context(),
		event.ID,
		eventType,
		body,
		event.ProviderRef,
		event.AmountMicro,
		event.TxHash,
		event.Confirmations,
	); err != nil {
		slog.Error("billing crypto webhook failed", "error", err, "event_id", event.ID)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "webhook processing failed")
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		slog.Error("billing crypto webhook response write failed", "error", err)
	}
}

func (h *CryptoWebhookHandlers) verifySignature(body []byte, r *http.Request) bool {
	provider := payment.NormalizeCryptoProvider(r.Header.Get("X-Crypto-Provider"))
	switch provider {
	case payment.CryptoProviderBTCPay:
		return payment.VerifyBTCPayWebhookSignature(body, r.Header.Get("BTCPay-Sig"), h.BTCPayWebhookSecret)
	case payment.CryptoProviderCryptomus:
		return payment.VerifyCryptomusWebhookSignature(body, r.Header.Get("sign"), h.CryptomusAPIKey)
	default:
		sig := r.Header.Get("Crypto-Signature")
		if sig == "" {
			sig = r.Header.Get("BTCPay-Sig")
		}
		secret := h.CryptoWebhookSecret
		if secret == "" {
			secret = h.BTCPayWebhookSecret
		}
		if secret == "" {
			return false
		}
		return payment.VerifyBTCPayWebhookSignature(body, sig, secret) ||
			payment.VerifyCryptomusWebhookSignature(body, r.Header.Get("sign"), h.CryptomusAPIKey)
	}
}

func (h *CryptoWebhookHandlers) SecretsConfigured() bool {
	if h == nil {
		return false
	}
	return strings.TrimSpace(h.CryptoWebhookSecret) != "" ||
		strings.TrimSpace(h.BTCPayWebhookSecret) != "" ||
		strings.TrimSpace(h.CryptomusAPIKey) != ""
}
