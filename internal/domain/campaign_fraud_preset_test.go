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

	_, _, _, _, ok = ResolveFraudPreset("unknown")
	require.False(t, ok)
}
