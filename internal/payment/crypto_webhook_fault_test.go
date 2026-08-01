package payment_test

import (
	"espx/pkg/faultproof"

	"context"
	"encoding/json"
	"sync"
	"testing"

	"espx/internal/payment"
	"espx/internal/paymenttest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_CryptoWebhookStormIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := paymenttest.SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := payment.NewService(infra.Pool, payment.NewProvider(infra.Cfg), infra.Cfg)

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

	const stormSize = 50
	var wg sync.WaitGroup
	wg.Add(stormSize)

	for i := 0; i < stormSize; i++ {
		go func() {
			defer wg.Done()
			_ = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, res.Intent.ProviderRef, amountMicro, "0xabc123", 12)
		}()
	}
	wg.Wait()

	intentID := uuid.MustParse(res.Intent.ID)
	intent, err := svc.GetPaymentIntent(ctx, intentID)
	require.NoError(t, err)
	require.Equal(t, "PAYMENT_INTENT_STATUS_SUCCEEDED", intent.Status)

	var holdCount int
	err = infra.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.crypto_holds WHERE payment_intent_id = $1
	`, intentID).Scan(&holdCount)
	require.NoError(t, err)
	require.Equal(t, 1, holdCount, "exactly one hold must be created despite the webhook storm")

	faultproof.Log(t, "crypto_webhook_storm", map[string]string{"idempotent": "true"})
}

func TestFault_CryptoWebhookReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := paymenttest.SetupPaymentFaultInfra(t)
	defer cleanup()

	ctx := context.Background()

	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 12

	svc := payment.NewService(infra.Pool, payment.NewProvider(infra.Cfg), infra.Cfg)

	customerID := uuid.New()
	paymenttest.SeedCustomer(t, infra.Pool, customerID)

	idempotencyKey := "crypto-replay-" + uuid.New().String()
	amountMicro := int64(50_000_000)

	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider": "crypto",
	})
	require.NoError(t, err)
	intentID := uuid.MustParse(res.Intent.ID)
	providerRef := res.Intent.ProviderRef

	eventID := "evt_crypto_replay_" + uuid.New().String()
	evtPayload := cryptoWebhookPayload{
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
		WHERE payment_intent_id = $1`, intentID)
	require.NoError(t, err)

	holdWorker := payment.NewCryptoHoldWorker(infra.Pool, infra.Cfg)
	require.NoError(t, holdWorker.ProcessHolds(ctx))

	outboxWorker := paymenttest.NewOutboxWorkerForFault(infra)
	for i := 0; i < replays; i++ {
		_, _ = outboxWorker.ProcessOutbox(ctx, 10)
	}

	seed := paymenttest.SeededPayment{
		CustomerID:  customerID,
		IntentID:    intentID,
		AmountMicro: amountMicro,
		ProviderRef: providerRef,
	}
	paymenttest.AssertPaymentFaultInvariants(t, infra.Pool, seed, amountMicro, 1)

	faultproof.Log(t, "crypto_webhook_replay", map[string]string{
		"subsystem":     "payment_crypto_webhook",
		"replays":       paymenttest.ItoaPaymentFault(replays),
		"proposal_rows": "1",
		"ledger_rows":   "1",
		"baseline_ok":   "true",
	})
}
