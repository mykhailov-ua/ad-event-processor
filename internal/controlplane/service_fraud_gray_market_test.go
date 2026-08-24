package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUpdateCampaignFraud_grayMarketPreset_appliesGMAFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: gray_market preset applies GMA flags (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	svc := NewService(context.Background(), pool, nil, nil, nil)
	defer svc.Close()
	ctx := context.Background()

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Gray Market Cust", 10_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       custID,
		Name:             "Gray Market Camp",
		BudgetLimitMicro: 10_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "gray-market-camp-1",
	})
	require.NoError(t, err)

	preset := domain.FraudPresetGrayMarket
	_, err = svc.UpdateCampaignFraudConfig(ctx, campID, CampaignFraudConfigUpdate{Preset: &preset})
	require.NoError(t, err)

	var (
		safePage, attestation, l15, tls, l1, linkSign bool
		attTTL                                        int32
		pass, block                                   int16
	)
	err = pool.QueryRow(ctx, `
		SELECT safe_page_enabled, attestation_enabled, attestation_ttl_sec,
		 l15_proxy_vpn_block_enabled, tls_fingerprint_block_enabled,
		 l1_cidr_block_enabled, link_signing_enabled,
		 fraud_threshold_pass, fraud_threshold_block
		FROM campaigns WHERE id = $1`, campID).Scan(
		&safePage, &attestation, &attTTL, &l15, &tls, &l1, &linkSign, &pass, &block,
	)
	require.NoError(t, err)
	require.True(t, safePage)
	require.True(t, attestation)
	require.GreaterOrEqual(t, attTTL, int32(60))
	require.True(t, l15)
	require.True(t, tls)
	require.True(t, l1)
	require.True(t, linkSign)
	require.Equal(t, int16(20), pass)
	require.Equal(t, int16(85), block)

	row, err := svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	camp := domain.CampaignFromDBRow(row)
	require.True(t, camp.SafePageEnabled)
	require.True(t, camp.AttestationEnabled)
	require.True(t, camp.L15ProxyVPNBlockEnabled)
	require.True(t, camp.TLSFingerprintBlockEnabled)
}

func TestResolveFraudPresetThresholds_grayMarket(t *testing.T) {
	svc := &Service{}
	pass, suspect, ivt, block, err := svc.resolveFraudPresetThresholds(t.Context(), domain.FraudPresetGrayMarket)
	require.NoError(t, err)
	require.Equal(t, uint8(20), pass)
	require.Equal(t, uint8(45), suspect)
	require.Equal(t, uint8(65), ivt)
	require.Equal(t, uint8(85), block)
}
