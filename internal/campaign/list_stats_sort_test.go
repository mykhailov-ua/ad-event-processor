package campaign

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCampaignListStatsSortField(t *testing.T) {
	t.Parallel()
	require.True(t, IsCampaignListStatsSortField("clicks"))
	require.True(t, IsCampaignListStatsSortField(" impressions "))
	require.False(t, IsCampaignListStatsSortField("ctr"))
	require.False(t, IsCampaignListStatsSortField("name"))
}

func TestIsCampaignListExtendedMetricSortField(t *testing.T) {
	t.Parallel()
	require.True(t, IsCampaignListExtendedMetricSortField("unique_clicks"))
	require.True(t, IsCampaignListExtendedMetricSortField("roi"))
	require.True(t, IsCampaignListExtendedMetricSortField("lp_ctr"))
	require.True(t, IsCampaignListExtendedMetricSortField("hold_leads"))
	require.False(t, IsCampaignListExtendedMetricSortField("clicks"))
}

func TestParseListSort_metadataFields(t *testing.T) {
	t.Parallel()
	allowed := CampaignListAllowedSortFields()
	req := httptest.NewRequest("GET", "/api/v1/campaigns?sort=budget_pct&order=desc", nil)
	field, order, err := parseListSort(req, allowed, "name")
	require.NoError(t, err)
	require.Equal(t, "budget_pct", field)
	require.Equal(t, "desc", order)

	req = httptest.NewRequest("GET", "/api/v1/campaigns?sort=group&order=asc", nil)
	field, order, err = parseListSort(req, allowed, "name")
	require.NoError(t, err)
	require.Equal(t, "group", field)
	require.Equal(t, "asc", order)
}
