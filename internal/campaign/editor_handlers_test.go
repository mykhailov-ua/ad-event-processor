package campaign

import (
	"context"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCampaignPatch_invalidPacingMode(t *testing.T) {
	t.Parallel()
	mode := "turbo"
	resp := validateCampaignPatch(context.Background(), uuid.New(), PatchCampaignRequest{PacingMode: &mode})
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.FieldErrors, "pacing_mode")
}

func TestValidateCampaignPatch_maskedBudgetDenied(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{Mask: authz.MaskMasked})
	budget := "100.00"
	resp := validateCampaignPatch(ctx, uuid.New(), PatchCampaignRequest{BudgetLimit: &budget})
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.FieldErrors, "budget_limit")
}

func TestFilterAndSortCampaigns_queryAndSort(t *testing.T) {
	t.Parallel()
	items := []CampaignDTO{
		{ID: "b", Name: "Beta", UpdatedAt: "2026-01-02T00:00:00Z"},
		{ID: "a", Name: "Alpha", UpdatedAt: "2026-01-01T00:00:00Z"},
	}
	got := filterAndSortCampaigns(items, "alp", "name", "asc", "")
	require.Len(t, got, 1)
	assert.Equal(t, "Alpha", got[0].Name)
}
