package fraudadmin

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyFraudScoringOverride_requiresTarget(t *testing.T) {
	err := ApplyFraudScoringOverride(t.Context(), nil, FraudScoringOverrideRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one of campaign_id or ip is required")
}

func TestApplyFraudScoringOverrideForCustomer_requiresCustomer(t *testing.T) {
	campaignID := uuid.New().String()
	err := ApplyFraudScoringOverrideForCustomer(t.Context(), nil, uuid.Nil, FraudOverrideRequest{
		CampaignID: &campaignID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "customer_id is required")
}

func TestApplyFraudScoringOverrideForCustomer_requiresTarget(t *testing.T) {
	err := ApplyFraudScoringOverrideForCustomer(t.Context(), nil, uuid.New(), FraudOverrideRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one of campaign_id, ip, or ip_hash is required")
}

func TestApplyFraudScoringOverrideForCustomer_invalidCampaignID(t *testing.T) {
	bad := "not-a-uuid"
	err := ApplyFraudScoringOverrideForCustomer(t.Context(), nil, uuid.New(), FraudOverrideRequest{
		CampaignID: &bad,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid campaign_id")
}

func TestResolveBlacklistIPByHash_requiresPool(t *testing.T) {
	_, err := resolveBlacklistIPByHash(t.Context(), nil, "0123456789abcdef0123456789abcdef")
	require.Error(t, err)
	require.Contains(t, err.Error(), "postgres pool not configured")
}
