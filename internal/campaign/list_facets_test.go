package campaign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCampaignListFacetsResponse_emptySlices(t *testing.T) {
	resp := CampaignListFacetsResponse{
		Countries: []string{},
		Owners:    []CampaignListFacetOwner{},
	}
	require.NotNil(t, resp.Countries)
	require.NotNil(t, resp.Owners)
	require.Empty(t, resp.Countries)
	require.Empty(t, resp.Owners)
}

func TestCampaignListFacetOwner_omitsEmptyEmailJSON(t *testing.T) {
	owner := CampaignListFacetOwner{UserID: "00000000-0000-4000-8000-000000000001"}
	require.Empty(t, owner.Email)
}
