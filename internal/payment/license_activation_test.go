package payment_test

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCryptoWebhook_LicenseActivation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	infra, cleanup := SetupPaymentFaultInfra(t)
	defer cleanup()
	dbtest.ApplyMigrations(t, infra.Pool, filepathJoinMigrations("internal/ledger/migrations"))

	ctx := context.Background()
	infra.Cfg.CryptoWebhookSecret = "crypto_test_secret"
	infra.Cfg.CryptoMinPaymentMicro = 10_000_000
	infra.Cfg.CryptoConfirmationDepth = 3

	svc := payment.NewService(infra.Pool, infra.Cfg)
	customerID := uuid.New()
	deploymentID := uuid.New()
	licenseID := uuid.New()

	_, err := infra.Pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'Lic Cust', 0, 'USD')
	`, customerID)
	require.NoError(t, err)

	idempotencyKey := "lic-idemp-" + uuid.New().String()
	amountMicro := int64(50_000_000)
	res, err := svc.CreatePaymentIntent(ctx, customerID, amountMicro, "USDT", idempotencyKey, map[string]string{
		"provider":      "crypto",
		"purpose":       "license_activation",
		"deployment_id": deploymentID.String(),
		"license_id":    licenseID.String(),
		"plan_code":     "enterprise",
	})
	require.NoError(t, err)

	providerRef := res.Intent.ProviderRef
	eventID := "evt_lic_" + uuid.New().String()
	bodyBytes, err := json.Marshal(map[string]any{
		"id":            eventID,
		"type":          "payment.succeeded",
		"tx_hash":       "0xlic",
		"amount_micro":  amountMicro,
		"provider_ref":  providerRef,
		"confirmations": 3,
	})
	require.NoError(t, err)

	err = svc.ProcessCryptoWebhook(ctx, eventID, "payment.succeeded", bodyBytes, providerRef, amountMicro, "0xlic", 3)
	require.NoError(t, err)

	var state string
	err = infra.Pool.QueryRow(ctx, `SELECT state FROM billing.license_status WHERE deployment_id = $1`, deploymentID).Scan(&state)
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", state)
}
