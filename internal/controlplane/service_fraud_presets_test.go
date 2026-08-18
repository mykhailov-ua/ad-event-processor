package controlplane

import (
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestDefaultFraudPolicyPresetDTOs_matchesDomain(t *testing.T) {
	out := defaultFraudPolicyPresetDTOs()
	require.Len(t, out, 3)
	byName := map[string]FraudPolicyPresetDTO{}
	for _, preset := range out {
		byName[preset.Name] = preset
	}
	pass, suspect, ivt, block, ok := domain.ResolveFraudPreset(domain.FraudPresetAggressive)
	require.True(t, ok)
	require.Equal(t, pass, byName[domain.FraudPresetAggressive].Pass)
	require.Equal(t, block, byName[domain.FraudPresetAggressive].Block)
	require.Equal(t, suspect, byName[domain.FraudPresetAggressive].Suspect)
	require.Equal(t, ivt, byName[domain.FraudPresetAggressive].IVT)
}

func TestResolveFraudPresetThresholds_fallbackWithoutPool(t *testing.T) {
	svc := &Service{}
	pass, suspect, ivt, block, err := svc.resolveFraudPresetThresholds(t.Context(), "balanced")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultFraudThresholdPass, pass)
	require.Equal(t, domain.DefaultFraudThresholdBlock, block)
	require.Equal(t, domain.DefaultFraudThresholdSuspect, suspect)
	require.Equal(t, domain.DefaultFraudThresholdIVT, ivt)
}

func TestResolveProposedFraudThresholds_preset(t *testing.T) {
	svc := &Service{}
	current := campaignFraudThresholds{pass: 30, suspect: 60, ivt: 80, block: 100}
	preset := "aggressive"
	out, err := svc.resolveProposedFraudThresholds(t.Context(), current, PreviewCampaignFraudRequest{Preset: &preset})
	require.NoError(t, err)
	require.Equal(t, uint8(20), out.pass)
	require.Equal(t, uint8(85), out.block)
}

func TestInvalidateFraudPolicyPresetCache(t *testing.T) {
	fraudPolicyPresetCache.mu.Lock()
	fraudPolicyPresetCache.loadedAt = time.Now()
	fraudPolicyPresetCache.rows = []fraudPolicyPresetRow{{name: "balanced"}}
	fraudPolicyPresetCache.mu.Unlock()

	invalidateFraudPolicyPresetCache()

	fraudPolicyPresetCache.mu.RLock()
	defer fraudPolicyPresetCache.mu.RUnlock()
	require.True(t, fraudPolicyPresetCache.loadedAt.IsZero())
	require.Nil(t, fraudPolicyPresetCache.rows)
}
