package fraudadmin

import (
	"context"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type presetThresholdHost struct{}

func (h presetThresholdHost) ConfigPool() *pgxpool.Pool { return nil }
func (h presetThresholdHost) ConfigClickHouse() *database.ClickHouseQuery { return nil }
func (h presetThresholdHost) ConfigActorID(context.Context) uuid.UUID    { return uuid.Nil }
func (h presetThresholdHost) ConfigAuditUpdate(context.Context, db.Querier, uuid.UUID, uuid.UUID, CampaignFraudAuditChange) {
}
func (h presetThresholdHost) ConfigResolvePresetThresholds(ctx context.Context, name string) (uint8, uint8, uint8, uint8, error) {
	return ResolvePresetThresholds(ctx, nil, name)
}
func (h presetThresholdHost) ConfigEnqueueUpdateCampaignFraud(context.Context, db.Querier, uuid.UUID) error {
	return nil
}

func TestResolvePresetThresholds_fallbackWithoutPool(t *testing.T) {
	pass, suspect, ivt, block, err := ResolvePresetThresholds(t.Context(), nil, "balanced")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultFraudThresholdPass, pass)
	require.Equal(t, domain.DefaultFraudThresholdBlock, block)
	require.Equal(t, domain.DefaultFraudThresholdSuspect, suspect)
	require.Equal(t, domain.DefaultFraudThresholdIVT, ivt)
}

func TestResolveProposedFraudThresholds_preset(t *testing.T) {
	host := presetThresholdHost{}
	current := campaignFraudThresholds{pass: 30, suspect: 60, ivt: 80, block: 100}
	preset := "aggressive"
	out, err := resolveProposedFraudThresholds(t.Context(), host, current, campaign.PreviewCampaignFraudRequest{Preset: &preset})
	require.NoError(t, err)
	require.Equal(t, uint8(20), out.pass)
	require.Equal(t, uint8(85), out.block)
}

func TestResolvePresetThresholds_enhancedDefense(t *testing.T) {
	pass, suspect, ivt, block, err := ResolvePresetThresholds(t.Context(), nil, domain.FraudPresetEnhancedDefense)
	require.NoError(t, err)
	require.Equal(t, uint8(20), pass)
	require.Equal(t, uint8(45), suspect)
	require.Equal(t, uint8(65), ivt)
	require.Equal(t, uint8(85), block)
}

func TestResolvePresetThresholds_socialInApp(t *testing.T) {
	pass, suspect, ivt, block, err := ResolvePresetThresholds(t.Context(), nil, domain.FraudPresetSocialInApp)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultFraudThresholdPass, pass)
	require.Equal(t, domain.DefaultFraudThresholdSuspect, suspect)
	require.Equal(t, domain.DefaultFraudThresholdIVT, ivt)
	require.Equal(t, domain.DefaultFraudThresholdBlock, block)
}

func TestDefaultFraudPolicyPresetDTOs_matchesDomain(t *testing.T) {
	out := DefaultPolicyPresetDTOs()
	require.Len(t, out, 5)
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
	legacy := byName[domain.FraudPresetEnhancedDefenseLegacy]
	require.Equal(t, uint8(20), legacy.Pass)
	require.Equal(t, uint8(85), legacy.Block)
	require.Equal(t, uint8(45), legacy.Suspect)
	require.Equal(t, uint8(65), legacy.IVT)
}
