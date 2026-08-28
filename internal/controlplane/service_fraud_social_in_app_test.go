package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUpdateCampaignFraud_socialInAppPreset_appliesFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: social_in_app preset applies flags (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	svc := NewService(context.Background(), pool, nil, nil, nil)
	defer svc.Close()
	ctx := context.Background()

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Social In-App Cust", 10_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Social In-App Camp",
		BudgetLimitMicro: 10_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "social-in-app-camp-1",
	})
	require.NoError(t, err)

	preset := domain.FraudPresetSocialInApp
	_, err = svc.UpdateCampaignFraudConfig(ctx, campID, campaign.PatchCampaignFraudRequest{Preset: &preset})
	require.NoError(t, err)

	var (
		socialInApp, proxyVPNBlock, tls bool
		connType                        string
		pass, block                     int16
	)
	err = pool.QueryRow(ctx, `
		SELECT social_in_app_enabled, proxy_vpn_block_enabled,
		 tls_fingerprint_block_enabled, conn_type_policy,
		 fraud_threshold_pass, fraud_threshold_block
		FROM campaigns WHERE id = $1`, campID).Scan(
		&socialInApp, &proxyVPNBlock, &tls, &connType, &pass, &block,
	)
	require.NoError(t, err)
	require.True(t, socialInApp)
	require.True(t, proxyVPNBlock)
	require.True(t, tls)
	require.Equal(t, string(domain.ConnTypeMobileOnly), connType)
	require.Equal(t, string(domain.SocialInAppConnTypePolicy), connType)
	require.Equal(t, int16(30), pass)
	require.Equal(t, int16(100), block)

	row, err := svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	camp := domain.CampaignFromDBRow(row)
	require.True(t, camp.SocialInAppEnabled)
	require.True(t, camp.ProxyVPNBlockEnabled)
	require.True(t, camp.TLSFingerprintBlockEnabled)
	require.Equal(t, domain.ConnTypeMobileOnly, camp.ConnTypePolicy)
	require.Equal(t, domain.SocialInAppConnTypePolicy, camp.ConnTypePolicy)
}
