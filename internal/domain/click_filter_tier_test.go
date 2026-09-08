package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResolveClickFilterTier_redirectOnlyUnlicensedEscalatesToFull(t *testing.T) {
	camp := &Campaign{ClickFilterTier: string(ClickFilterTierRedirectOnly)}
	got := ResolveClickFilterTier(camp, "full", false)
	assert.Equal(t, ClickFilterTierFull, got)
}

func TestResolveClickFilterTier_fraudFeaturesEscalateToFull(t *testing.T) {
	camp := &Campaign{
		ClickFilterTier:     string(ClickFilterTierLight),
		SilentRejectEnabled: true,
	}
	got := ResolveClickFilterTier(camp, "full", true)
	assert.Equal(t, ClickFilterTierFull, got)
}

func TestResolveClickFilterTier_envDefaultLight(t *testing.T) {
	got := ResolveClickFilterTier(nil, "light", true)
	assert.Equal(t, ClickFilterTierLight, got)
}

func TestCampaignRequiresFullClickFilters_segment(t *testing.T) {
	camp := &Campaign{SegmentIncludeID: uuid.New()}
	assert.True(t, CampaignRequiresFullClickFilters(camp))
}

func TestValidateClickFilterTierForSave_redirectOnlyUnlicensed(t *testing.T) {
	err := ValidateClickFilterTierForSave("redirect_only", false)
	assert.ErrorIs(t, err, ErrClickFilterTierRedirectOnlyUnlicensed)
}
