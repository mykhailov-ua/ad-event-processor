package controlplane

import (
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePatchConnTypePolicy(t *testing.T) {
	t.Parallel()
	got, ok, err := campaign.ParsePatchConnTypePolicy(ptrString("mobile_only"))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "mobile_only", got)

	_, _, err = campaign.ParsePatchConnTypePolicy(ptrString("bogus"))
	require.Error(t, err)
}

func TestParsePatchLinkSigningTTLSec(t *testing.T) {
	t.Parallel()
	v := int32(900)
	got, ok, err := campaign.ParsePatchLinkSigningTTLSec(&v)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int32(900), got)

	bad := int32(30)
	_, _, err = campaign.ParsePatchLinkSigningTTLSec(&bad)
	require.Error(t, err)
}

func ptrString(s string) *string { return &s }

func TestResolvePatchBudgetLimitMicro(t *testing.T) {
	t.Parallel()
	micro := int64(50_000_000)
	got, err := campaign.ResolvePatchBudgetLimitMicro(campaign.PatchCampaignRequest{BudgetLimitMicro: &micro})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int64(50_000_000), *got)

	legacy := "25.50"
	got, err = campaign.ResolvePatchBudgetLimitMicro(campaign.PatchCampaignRequest{BudgetLimit: &legacy})
	require.NoError(t, err)
	require.Equal(t, int64(25_500_000), *got)

	_, err = campaign.ResolvePatchBudgetLimitMicro(campaign.PatchCampaignRequest{})
	require.NoError(t, err)

	bad := int64(0)
	_, err = campaign.ResolvePatchBudgetLimitMicro(campaign.PatchCampaignRequest{BudgetLimitMicro: &bad})
	require.Error(t, err)
}

func TestParsePatchStatus(t *testing.T) {
	t.Parallel()
	active := "active"
	got, set, err := campaign.ParsePatchStatus(&active)
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, db.CampaignStatusTypeACTIVE, got)

	paused := "PAUSED"
	got, set, err = campaign.ParsePatchStatus(&paused)
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, db.CampaignStatusTypePAUSED, got)

	_, set, err = campaign.ParsePatchStatus(nil)
	require.NoError(t, err)
	require.False(t, set)

	bad := "DRAINING"
	_, _, err = campaign.ParsePatchStatus(&bad)
	require.Error(t, err)

	archived := "ARCHIVED"
	got, set, err = campaign.ParsePatchStatus(&archived)
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, db.CampaignStatusTypeARCHIVED, got)
}
