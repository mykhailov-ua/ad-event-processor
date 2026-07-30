package payment

import (
	"espx/pkg/faultproof"

	"context"
	"encoding/json"
	"sync"
	"testing"

	"espx/internal/payment/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_CryptoWebhookStormIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := NewService(infra.Pool, NewProvider(infra.Cfg), infra.Cfg)

	customerID := uuid.New()
	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Crypto Customer', 0.00, 'USD')
	`, customerID)
	require.NoError(t, err)

	idempotencyKey := "crypto-storm-" + uuid.New().String()
	amountMicro := int64(50_000_000)

	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)

	eventID := "evt_crypto_storm_" + uuid.New().String()
	evtPayload := cryptoEvent{
		ID:            eventID,
		Type:          "payment.succeeded",
		TxHash:        "0xabc123",
		AmountMicro:   amountMicro,
		Currency:      "USDT",
		Confirmations: 12,
		ProviderRef:   res.Intent.ProviderRef.String,
	}
	bodyBytes, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	const stormSize = 50
	var wg sync.WaitGroup
	wg.Add(stormSize)

	for i := 0; i < stormSize; i++ {
		go func() {
			defer wg.Done()
			_ = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, res.Intent.ProviderRef.String, amountMicro, "0xabc123", 12)
		}()
	}
	wg.Wait()

	intent, err := svc.GetPaymentIntent(ctx, uuid.UUID(res.Intent.ID.Bytes))
	require.NoError(t, err)
	require.Equal(t, db.PaymentPaymentIntentStatusSUCCEEDED, intent.Status)

	var holdCount int
	err = infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.crypto_holds WHERE payment_intent_id = $1
	`, intent.ID).Scan(&holdCount)
	require.NoError(t, err)
	require.Equal(t, 1, holdCount, "exactly one hold must be created despite the webhook storm")

	faultproof.Log(t, "crypto_webhook_storm", map[string]string{"idempotent": "true"})
}

func TestFault_CryptoWebhookReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := NewService(infra.Pool, NewProvider(infra.Cfg), infra.Cfg)

	customerID := uuid.New()
	seedCustomer(t, infra.Pool, customerID)

	idempotencyKey := "crypto-replay-" + uuid.New().String()
	amountMicro := int64(50_000_000)

	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)
	intentID := uuid.UUID(res.Intent.ID.Bytes)
	providerRef := res.Intent.ProviderRef.String

	eventID := "evt_crypto_replay_" + uuid.New().String()
	evtPayload := cryptoEvent{
		ID:            eventID,
		Type:          "payment.succeeded",
		TxHash:        "0xreplay123",
		AmountMicro:   amountMicro,
		Currency:      "USDT",
		Confirmations: 12,
		ProviderRef:   providerRef,
	}
	bodyBytes, err := json.Marshal(evtPayload)
	require.NoError(t, err)

	const replays = 3
	for i := 0; i < replays; i++ {
		err = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, providerRef, amountMicro, "0xreplay123", 12)
		require.NoError(t, err)
	}

	var webhookRows int
	require.NoError(t, infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.webhook_events
		WHERE provider = 'crypto' AND provider_event_id = $1`, eventID).Scan(&webhookRows))
	require.Equal(t, 1, webhookRows)

	_, err = infra.Pool.Exec(ctx, `
		UPDATE payment.crypto_holds SET release_at = now() - interval '1 second'
		WHERE payment_intent_id = $1`, res.Intent.ID)
	require.NoError(t, err)

	holdWorker := NewCryptoHoldWorker(infra.Pool, infra.Cfg)
	require.NoError(t, holdWorker.ProcessHolds(ctx))

	outboxWorker := newOutboxWorkerForFault(infra)
	for i := 0; i < replays; i++ {
		_, _ = outboxWorker.ProcessOutbox(ctx, 10)
	}

	seed := seededPayment{
		CustomerID:  customerID,
		IntentID:    intentID,
		AmountMicro: amountMicro,
		ProviderRef: providerRef,
	}
	assertPaymentFaultInvariants(t, infra.Pool, seed, amountMicro, 1)

	faultproof.Log(t, "crypto_webhook_replay", map[string]string{
		"subsystem":     "payment_crypto_webhook",
		"replays":       itoaPaymentFault(replays),
		"proposal_rows": "1",
		"ledger_rows":   "1",
		"baseline_ok":   "true",
	})
}
