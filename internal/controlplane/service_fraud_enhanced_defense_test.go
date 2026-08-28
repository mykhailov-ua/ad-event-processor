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

func TestUpdateCampaignFraud_enhancedDefensePreset_appliesDefenseFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: enhanced_defense preset applies defense flags (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	svc := NewService(context.Background(), pool, nil, nil, nil)
	defer svc.Close()
	ctx := context.Background()

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Enhanced Defense Cust", 10_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Enhanced Defense Camp",
		BudgetLimitMicro: 10_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "enhanced-defense-camp-1",
	})
	require.NoError(t, err)

	preset := domain.FraudPresetEnhancedDefense
	_, err = svc.UpdateCampaignFraudConfig(ctx, campID, campaign.PatchCampaignFraudRequest{Preset: &preset})
	require.NoError(t, err)

	var (
		safePage, silentReject, attestation, proxyVPNBlock, tls, cidrBlock, linkSign bool
		attTTL                                                                       int32
		clickDelivery                                                                string
		pass, block                                                                  int16
	)
	err = pool.QueryRow(ctx, `
		SELECT safe_page_enabled, silent_reject_enabled, attestation_enabled, attestation_ttl_sec,
		 proxy_vpn_block_enabled, tls_fingerprint_block_enabled,
		 cidr_block_enabled, link_signing_enabled, click_delivery,
		 fraud_threshold_pass, fraud_threshold_block
		FROM campaigns WHERE id = $1`, campID).Scan(
		&safePage, &silentReject, &attestation, &attTTL, &proxyVPNBlock, &tls, &cidrBlock, &linkSign, &clickDelivery, &pass, &block,
	)
	require.NoError(t, err)
	require.True(t, safePage)
	require.True(t, silentReject)
	require.Equal(t, "redirect", clickDelivery)
	require.True(t, attestation)
	require.GreaterOrEqual(t, attTTL, int32(60))
	require.True(t, proxyVPNBlock)
	require.True(t, tls)
	require.True(t, cidrBlock)
	require.True(t, linkSign)
	require.Equal(t, int16(20), pass)
	require.Equal(t, int16(85), block)

	row, err := svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	camp := domain.CampaignFromDBRow(row)
	require.True(t, camp.SafePageEnabled)
	require.True(t, camp.SilentRejectEnabled)
	require.Equal(t, "redirect", camp.ClickDelivery)
	require.True(t, camp.AttestationEnabled)
	require.True(t, camp.ProxyVPNBlockEnabled)
	require.True(t, camp.TLSFingerprintBlockEnabled)
}
