package controlplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOpenAPI_billingInvariantSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/billing.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["BillingInvariant"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto billingadmin.InvariantDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "billingadmin.InvariantDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_billingStatementSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/billing.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["BillingStatement"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto billingadmin.StatementDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "billingadmin.StatementDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_invoiceSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/billing.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["Invoice"]["properties"].(map[string]any)
	require.True(t, ok)

	inv := domain.Invoice{
		ID:           "00000000-0000-0000-0000-000000000001",
		CustomerID:   "00000000-0000-0000-0000-000000000002",
		BillingMonth: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Currency:     "USD",
	}
	sample, err := json.Marshal(inv)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "Invoice json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_walletSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/billing.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["Wallet"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto billingadmin.WalletDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "billingadmin.WalletDTO json field %q missing from OpenAPI schema", key)
	}
}
