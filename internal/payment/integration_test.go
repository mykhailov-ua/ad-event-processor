package payment_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	ads_db "ad-event-processor/internal/domain/db"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/payment/db"
	"ad-event-processor/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := dbtest.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := dbtest.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		PaymentInternalToken:    "payment_secret_token",
		SettlementInternalToken: "settlement_secret_token",
		StripeWebhookSecret:     "stripe_wh_secret",
		MaxRetries:              3,
	}
	cfg.Lifecycle.ShutdownTimeoutMs = 1000

	svc := payment.NewService(pool, cfg)

	ctx := context.Background()

	customerID := uuid.New()
	qAds := ads_db.New(pool)
	_, err := qAds.CreateCustomer(ctx, ads_db.CreateCustomerParams{
		ID:       ingestion.ToUUID(customerID),
		Name:     "Test Payment Customer",
		Balance:  0,
		Currency: "USD",
	})
	require.NoError(t, err)

	idempotencyKey := "idempotency_key_test_123"
	amountMicro := int64(50000000)

	result, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USD", idempotencyKey, map[string]string{"foo": "bar"})
	require.NoError(t, err)
	intent := result.Intent
	assert.Equal(t, "PAYMENT_INTENT_STATUS_PENDING_PROVIDER", intent.Status)
	assert.Equal(t, amountMicro, intent.AmountMicro)
	assert.NotEmpty(t, result.CheckoutURL)

	result2, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USD", idempotencyKey, map[string]string{"foo": "bar"})
	require.NoError(t, err)
	intent2 := result2.Intent
	assert.Equal(t, intent.ID, intent2.ID)

	_, err = svc.CreatePaymentIntent(ctx, customerID, amountMicro+10, "USD", idempotencyKey, map[string]string{"foo": "bar"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "idempotency key conflict")

	providerRef := intent.ProviderRef
	stripeCents, err := payment.MicroToStripeAmount(amountMicro)
	require.NoError(t, err)
	stripePayload := fmt.Sprintf(`{
		"id": "evt_stripe_test_999",
		"type": "payment_intent.succeeded",
		"data": {
			"object": {
				"id": "%s",
				"amount": %d
			}
		}
	}`, providerRef, stripeCents)

	err = svc.ProcessStripeWebhook(ctx, "evt_stripe_test_999", "payment_intent.succeeded", []byte(stripePayload), providerRef, amountMicro, stripePayload)
	require.NoError(t, err)

	intentID, err := uuid.Parse(intent.ID)
	require.NoError(t, err)
	intentUpdated, err := svc.GetPaymentIntent(ctx, intentID)
	require.NoError(t, err)
	assert.Equal(t, "PAYMENT_INTENT_STATUS_SUCCEEDED", intentUpdated.Status)

	qPayment := db.New(pool)
	outboxEvents, err := qPayment.GetPendingOutboxEventsForUpdate(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outboxEvents, 1)
	assert.Equal(t, "SETTLE_BALANCE", outboxEvents[0].EventType)

	redisShards := []redis.UniversalClient{redisClient}
	mgmtSvc := controlplane.NewService(context.Background(), pool, redisShards, ingestion.NewStaticSlotSharder(len(redisShards)), cfg)
	settleHandler := controlplane.NewSettlementHandler(mgmtSvc, cfg)

	outboxWorker := payment.NewOutboxWorker(pool, cfg)
	outboxWorker.SetSettlementAPI(settleHandler.PaymentSettlement())
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	go outboxWorker.Start(ctxCancel, 50*time.Millisecond)

	require.Eventually(t, func() bool {
		events, err := db.New(pool).GetPendingOutboxEventsForUpdate(ctx, 10)
		return err == nil && len(events) == 0
	}, 5*time.Second, 100*time.Millisecond)

	customer, err := qAds.GetCustomerForUpdate(ctx, ingestion.ToUUID(customerID))
	require.NoError(t, err)
	assert.Equal(t, amountMicro, customer.Balance)

	ledgerRows, err := qAds.ListCustomerLedger(ctx, ads_db.ListCustomerLedgerParams{
		CustomerID: ingestion.ToUUID(customerID),
		Limit:      10,
		Offset:     0,
	})
	require.NoError(t, err)
	require.Len(t, ledgerRows, 1)
	assert.Equal(t, amountMicro, ledgerRows[0].Amount)
	assert.Equal(t, ads_db.LedgerType("PAYMENT_TOPUP"), ledgerRows[0].Type)
	assert.Equal(t, "payment:"+intentID.String(), ledgerRows[0].IdempotencyHash.String)
	assert.Equal(t, ingestion.ToUUID(intentID), ledgerRows[0].PaymentIntentID)

	refundMicro := amountMicro / 2
	refundID := "re_integration_" + uuid.New().String()
	refundCents, err := payment.MicroToStripeAmount(refundMicro)
	require.NoError(t, err)
	refundPayload := fmt.Sprintf(`{
		"id": "evt_refund_integration",
		"type": "refund.created",
		"data": {
			"object": {
				"id": "%s",
				"amount": %d,
				"payment_intent": "%s",
				"status": "succeeded"
			}
		}
	}`, refundID, refundCents, providerRef)
	err = svc.ProcessStripeRefundWebhook(ctx, "evt_refund_integration", "refund.created", []byte(refundPayload), refundID, providerRef, refundMicro, "succeeded")
	require.NoError(t, err)

	refundOutbox, err := qPayment.GetPendingOutboxEventsForUpdate(ctx, 10)
	require.NoError(t, err)
	require.Len(t, refundOutbox, 1)
	assert.Equal(t, payment.OutboxEventReverseBalance, refundOutbox[0].EventType)

	n, err := outboxWorker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	customerAfterRefund, err := qAds.GetCustomerForUpdate(ctx, ingestion.ToUUID(customerID))
	require.NoError(t, err)
	assert.Equal(t, amountMicro-refundMicro, customerAfterRefund.Balance)

	ledgerRows, err = qAds.ListCustomerLedger(ctx, ads_db.ListCustomerLedgerParams{
		CustomerID: ingestion.ToUUID(customerID),
		Limit:      10,
		Offset:     0,
	})
	require.NoError(t, err)
	require.Len(t, ledgerRows, 2)
	assert.Equal(t, ads_db.LedgerType("PAYMENT_REFUND"), ledgerRows[0].Type)
	assert.Equal(t, -refundMicro, ledgerRows[0].Amount)
}
