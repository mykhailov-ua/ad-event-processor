package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscrepancyBuySellRowsFromMaps(t *testing.T) {
	t.Parallel()
	rows := discrepancyBuySellRowsFromMaps([]map[string]any{{
		"campaign_id":     "c1",
		"buy_spend_micro": int64(100),
		"sell_rev_micro":  int64(150),
		"delta_micro":     int64(50),
		"delta_pct":       50.0,
	}})
	require.Len(t, rows, 1)
	require.Equal(t, "c1", rows[0].CampaignID)
	require.Equal(t, int64(50), rows[0].DeltaMicro)
}

func TestCustomerPortfolioSummaryFromDTO(t *testing.T) {
	t.Parallel()
	summary := customerPortfolioSummaryFromDTO(BuyerPortfolioDTO{
		Active:         2,
		Paused:         1,
		Impressions7d:  100,
		OverspendCount: 1,
		Attention:      []BuyerAttentionDTO{{ID: "a1"}},
		Campaigns:      []BuyerCampaignPortfolioRowDTO{{ID: "c1"}},
	})
	require.Equal(t, 2, summary.Active)
	require.Equal(t, 1, summary.AttentionCount)
	require.Equal(t, 1, summary.CampaignsSample)
}
