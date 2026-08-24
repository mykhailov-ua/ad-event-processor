package controlplane

import (
	"testing"

	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyFraudScoringOverride_requiresTarget(t *testing.T) {
	svc := &Service{}
	err := svc.ApplyFraudScoringOverride(t.Context(), FraudScoringOverrideRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one of campaign_id or ip is required")
}

func TestApplyFraudScoringOverrideForCustomer_requiresCustomer(t *testing.T) {
	svc := &Service{}
	campaignID := uuid.New().String()
	err := svc.ApplyFraudScoringOverrideForCustomer(t.Context(), uuid.Nil, FraudOverrideRequest{
		CampaignID: &campaignID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "customer_id is required")
}

func TestApplyFraudScoringOverrideForCustomer_requiresTarget(t *testing.T) {
	svc := &Service{}
	err := svc.ApplyFraudScoringOverrideForCustomer(t.Context(), uuid.New(), FraudOverrideRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one of campaign_id, ip, or ip_hash is required")
}

func TestApplyFraudScoringOverrideForCustomer_invalidCampaignID(t *testing.T) {
	bad := "not-a-uuid"
	svc := &Service{}
	err := svc.ApplyFraudScoringOverrideForCustomer(t.Context(), uuid.New(), FraudOverrideRequest{
		CampaignID: &bad,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid campaign_id")
}

func TestResolveBlacklistIPByHash_requiresPool(t *testing.T) {
	svc := &Service{cfg: testPIIConfig(t)}
	_, err := svc.resolveBlacklistIPByHash(t.Context(), "0123456789abcdef0123456789abcdef")
	require.Error(t, err)
	require.Contains(t, err.Error(), "postgres pool not configured")
}

func testPIIConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.PIISaltHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	cfg.PIISaltVersion = 1
	return cfg
}
