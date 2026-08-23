package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFraudPreset(t *testing.T) {
	pass, suspect, ivt, block, ok := ResolveFraudPreset(FraudPresetBalanced)
	require.True(t, ok)
	require.Equal(t, DefaultFraudThresholdPass, pass)
	require.Equal(t, DefaultFraudThresholdSuspect, suspect)
	require.Equal(t, DefaultFraudThresholdIVT, ivt)
	require.Equal(t, DefaultFraudThresholdBlock, block)

	pass, suspect, ivt, block, ok = ResolveFraudPreset(FraudPresetGrayMarket)
	require.True(t, ok)
	require.Equal(t, uint8(20), pass)
	require.Equal(t, uint8(45), suspect)
	require.Equal(t, uint8(65), ivt)
	require.Equal(t, uint8(85), block)
	require.True(t, IsGrayMarketFraudPreset("gray_market"))
	require.False(t, IsGrayMarketFraudPreset("aggressive"))

	_, _, _, _, ok = ResolveFraudPreset("unknown")
	require.False(t, ok)
}

func TestSocialInAppConnTypePolicy_mobileOnly(t *testing.T) {
	require.Equal(t, ConnTypeMobileOnly, SocialInAppConnTypePolicy)
	require.True(t, IsSocialInAppFraudPreset(FraudPresetSocialInApp))
}
