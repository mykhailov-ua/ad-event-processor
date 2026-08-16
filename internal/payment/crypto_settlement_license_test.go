package payment_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/payment"
	"github.com/bidshard/ad-event-processor/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCryptoGateway_SettlementAndLicenseActivation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()
	dbtest.ApplyMigrations(t, infra.Pool, filepathJoinMigrations("internal/ledger/migrations"))

	ctx := context.Background()
	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 1

	svc := payment.NewService(infra.Pool, infra.Cfg)
	customerID := uuid.New()
	deploymentID := uuid.New()
	licenseID := uuid.New()

	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'Settlement Lic', 0, 'USD')
	`, customerID)
	require.NoError(t, err)

	amountMicro := int64(50_000_000)
	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", "settle-lic-"+uuid.New().String(), map[string]string{
		"provider":      "crypto",
		"purpose":       "license_activation",
		"deployment_id": deploymentID.String(),
		"license_id":    licenseID.String(),
		"plan_code":     "enterprise",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.DepositAddress)

	providerRef := res.Intent.ProviderRef
	eventID := "evt_settle_" + uuid.New().String()
	bodyBytes, err := json.Marshal(map[string]any{
		"id":            eventID,
		"type":          "payment.succeeded",
		"tx_hash":       "0xsettle",
		"amount_micro":  amountMicro,
		"provider_ref":  providerRef,
		"confirmations": 1,
	})
	require.NoError(t, err)
	require.NoError(t, svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, providerRef, amountMicro, "0xsettle", 1))

	var licenseState string
	err = infra.Pool.QueryRow(ctx, `SELECT state FROM billing.license_status WHERE deployment_id = $1`, deploymentID).Scan(&licenseState)
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", licenseState)

	intentID := uuid.MustParse(res.Intent.ID)
	_, err = infra.Pool.Exec(ctx, `
		UPDATE payment.crypto_holds SET release_at = now() - interval '1 second' WHERE payment_intent_id = $1
	`, intentID)
	require.NoError(t, err)

	holdWorker := payment.NewCryptoHoldWorker(infra.Pool, infra.Cfg)
	require.NoError(t, holdWorker.ProcessHolds(ctx))

	outboxWorker := NewOutboxWorkerForFault(infra)
	n, err := outboxWorker.ProcessOutbox(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var balance int64
	err = infra.Pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, customerID).Scan(&balance)
	require.NoError(t, err)
	require.Equal(t, amountMicro, balance)

	var ledgerCount int
	err = infra.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM balance_ledger WHERE customer_id = $1`, customerID).Scan(&ledgerCount)
	require.NoError(t, err)
	require.Equal(t, 1, ledgerCount)
}
