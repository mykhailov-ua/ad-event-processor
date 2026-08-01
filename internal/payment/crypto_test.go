package payment_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"espx/internal/payment"
	"espx/internal/paymenttest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCryptoGateway_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := paymenttest.SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := payment.NewService(infra.Pool, infra.Cfg)

	customerID := uuid.New()

	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Crypto Customer', 0.00, 'USD')
	`, customerID)
	require.NoError(t, err)

	idempotencyKey := "crypto-idemp-" + uuid.New().String()
	amountMicro := int64(50_000_000)

	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)
	require.Equal(t, "crypto", res.Intent.Provider)
	require.Equal(t, "PAYMENT_INTENT_STATUS_PENDING_PROVIDER", res.Intent.Status)

	providerRef := res.Intent.ProviderRef
	require.NotEmpty(t, providerRef)

	eventID := "evt_crypto_" + uuid.New().String()
	evtPayload := cryptoWebhookPayload{
		ID:            eventID,
		Type:          "payment.succeeded",
		TxHash:        "0xabc123",
		AmountMicro:   amountMicro,
		Currency:      "USDT",
		Confirmations: 5,
		ProviderRef:   providerRef,
	}
	bodyBytes, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	err = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, providerRef, amountMicro, "0xabc123", 5)
	require.NoError(t, err)

	intentID := uuid.MustParse(res.Intent.ID)
	intent, err := svc.GetPaymentIntent(ctx, intentID)
	require.NoError(t, err)
	require.Equal(t, "PAYMENT_INTENT_STATUS_PROCESSING", intent.Status)

	var holdCount int
	err = infra.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM payment.crypto_holds WHERE payment_intent_id = $1`, intentID).Scan(&holdCount)
	require.NoError(t, err)
	require.Equal(t, 0, holdCount)

	eventID2 := "evt_crypto_" + uuid.New().String()
	evtPayload.ID = eventID2
	evtPayload.Confirmations = 12
	bodyBytes2, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	err = svc.ProcessCryptoWebhook(ctx, eventID2, "payment.succeeded", bodyBytes2, providerRef, amountMicro, "0xabc123", 12)
	require.NoError(t, err)

	intent, err = svc.GetPaymentIntent(ctx, intentID)
	require.NoError(t, err)
	require.Equal(t, "PAYMENT_INTENT_STATUS_SUCCEEDED", intent.Status)

	var holdStatus string
	var holdReleaseAt time.Time
	err = infra.Pool.QueryRow(ctx, `
		SELECT status, release_at FROM payment.crypto_holds WHERE payment_intent_id = $1
	`, intentID).Scan(&holdStatus, &holdReleaseAt)
	require.NoError(t, err)
	require.Equal(t, "HELD", holdStatus)
	require.True(t, holdReleaseAt.After(time.Now()))

	_, err = infra.Pool.Exec(ctx, `
		UPDATE payment.crypto_holds SET release_at = now() - interval '1 second' WHERE payment_intent_id = $1
	`, intentID)
	require.NoError(t, err)

	holdWorker := payment.NewCryptoHoldWorker(infra.Pool, infra.Cfg)
	err = holdWorker.ProcessHolds(ctx)
	require.NoError(t, err)

	err = infra.Pool.QueryRow(ctx, `
		SELECT status FROM payment.crypto_holds WHERE payment_intent_id = $1
	`, intentID).Scan(&holdStatus)
	require.NoError(t, err)
	require.Equal(t, "RELEASED", holdStatus)

	var outboxCount int
	err = infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.payment_outbox WHERE event_type = 'SETTLE_BALANCE'
	`).Scan(&outboxCount)
	require.NoError(t, err)
	require.Equal(t, 1, outboxCount)

	outboxWorker := paymenttest.NewOutboxWorkerForFault(infra)
	n, err := outboxWorker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var balance int64
	err = infra.Pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, customerID).Scan(&balance)
	require.NoError(t, err)
	require.Equal(t, int64(50_000_000), balance)
}

func TestCryptoGateway_UnderpayRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := paymenttest.SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := payment.NewService(infra.Pool, infra.Cfg)

	customerID := uuid.New()
	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Crypto Customer', 0.00, 'USD')
	`, customerID)
	require.NoError(t, err)

	idempotencyKey := "crypto-underpay-" + uuid.New().String()
	amountMicro := int64(50_000_000)

	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)

	eventID := "evt_crypto_" + uuid.New().String()
	evtPayload := cryptoWebhookPayload{
		ID:            eventID,
		Type:          "payment.succeeded",
		TxHash:        "0xabc123",
		AmountMicro:   40_000_000,
		Currency:      "USDT",
		Confirmations: 12,
		ProviderRef:   res.Intent.ProviderRef,
	}
	bodyBytes, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	err = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, res.Intent.ProviderRef, 40_000_000, "0xabc123", 12)
	require.NoError(t, err)

	intent, err := svc.GetPaymentIntent(ctx, uuid.MustParse(res.Intent.ID))
	require.NoError(t, err)
	require.Equal(t, "PAYMENT_INTENT_STATUS_FAILED", intent.Status)

	var holdCount int
	err = infra.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM payment.crypto_holds WHERE payment_intent_id = $1`, uuid.MustParse(res.Intent.ID)).Scan(&holdCount)
	require.NoError(t, err)
	require.Equal(t, 0, holdCount)
}

func TestCryptoGateway_FraudGateBlocksRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := paymenttest.SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := payment.NewService(infra.Pool, infra.Cfg)

	customerID := uuid.New()
	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Crypto Customer', 0.00, 'USD')
	`, customerID)
	require.NoError(t, err)

	idempotencyKey := "crypto-fraud-" + uuid.New().String()
	amountMicro := int64(50_000_000)

	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)

	eventID := "evt_crypto_" + uuid.New().String()
	evtPayload := cryptoWebhookPayload{
		ID:            eventID,
		Type:          "payment.succeeded",
		TxHash:        "0xabc123",
		AmountMicro:   amountMicro,
		Currency:      "USDT",
		Confirmations: 12,
		ProviderRef:   res.Intent.ProviderRef,
	}
	bodyBytes, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	err = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, res.Intent.ProviderRef, amountMicro, "0xabc123", 12)
	require.NoError(t, err)

	_, err = infra.Pool.Exec(ctx, `
		UPDATE payment.crypto_holds SET release_at = now() - interval '1 second' WHERE payment_intent_id = $1
	`, uuid.MustParse(res.Intent.ID))
	require.NoError(t, err)

	disputeID := uuid.New()
	_, err = infra.Pool.Exec(ctx, `
		INSERT INTO payment.payment_disputes (id, payment_intent_id, provider, provider_dispute_id, amount_micro, status)
		VALUES ($1, $2, 'crypto', 'disp_crypto_123', $3, 'OPEN')
	`, disputeID, uuid.MustParse(res.Intent.ID), amountMicro)
	require.NoError(t, err)

	holdWorker := payment.NewCryptoHoldWorker(infra.Pool, infra.Cfg)
	err = holdWorker.ProcessHolds(ctx)
	require.NoError(t, err)

	var holdStatus string
	err = infra.Pool.QueryRow(ctx, `
		SELECT status FROM payment.crypto_holds WHERE payment_intent_id = $1
	`, uuid.MustParse(res.Intent.ID)).Scan(&holdStatus)
	require.NoError(t, err)
	require.Equal(t, "FRAUD_BLOCKED", holdStatus)

	var outboxCount int
	err = infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.payment_outbox WHERE event_type = 'SETTLE_BALANCE'
	`).Scan(&outboxCount)
	require.NoError(t, err)
	require.Equal(t, 0, outboxCount)
}

func TestCryptoGateway_WebhookHTTPHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := paymenttest.SetupPaymentFaultInfra(t)
	defer cleanup()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := payment.NewService(infra.Pool, infra.Cfg)
	handler := payment.NewWebhookHandler(svc, infra.Cfg)

	customerID := uuid.New()
	_, err := infra.Pool.Exec(context.Background(), `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Crypto Customer', 0.00, 'USD')
	`, customerID)
	require.NoError(t, err)

	res, err := svc.CreatePaymentIntent(context.Background(), customerID, 50_000_000, "USDT", "crypto-http-idemp", map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)

	eventID := "evt_crypto_http"
	evtPayload := cryptoWebhookPayload{
		ID:            eventID,
		Type:          "payment.succeeded",
		TxHash:        "0xabc123",
		AmountMicro:   50_000_000,
		Currency:      "USDT",
		Confirmations: 12,
		ProviderRef:   res.Intent.ProviderRef,
	}
	bodyBytes, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	ts := time.Now().Unix()
	tsStr := fmt.Sprintf("%d", ts)
	mac := hmac.New(sha256.New, []byte("crypto_test_secret"))
	mac.Write([]byte(tsStr + "." + string(bodyBytes)))
	sigHeader := fmt.Sprintf("t=%s,v1=%s", tsStr, hex.EncodeToString(mac.Sum(nil)))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/crypto", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Crypto-Signature", sigHeader)

	rr := httptest.NewRecorder()
	handler.HandleCryptoWebhookForTest(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "OK", rr.Body.String())
}
