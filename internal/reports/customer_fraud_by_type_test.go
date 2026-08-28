package reports

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateCustomerFraudByType_groupsCategories(t *testing.T) {
	t.Parallel()
	rows := []FraudBreakdownRowDTO{
		{CampaignID: "c1", FraudReason: "tls_ja4_mismatch", EventCount: 100, SilentRejectCount: 20},
		{CampaignID: "c1", FraudReason: "datacenter_ip", EventCount: 50, SilentRejectCount: 5},
	}
	got := aggregateCustomerFraudByType(rows, "")
	require.Len(t, got, 2)
	var totalEvents int64
	for _, row := range got {
		totalEvents += row.EventCount
		assert.NotEmpty(t, row.FraudCategoryLabel)
	}
	assert.Equal(t, int64(150), totalEvents)
}

func TestAggregateCustomerFraudByType_categoryFilter(t *testing.T) {
	t.Parallel()
	rows := []FraudBreakdownRowDTO{
		{CampaignID: "c1", FraudReason: "tls_ja4_mismatch", EventCount: 100, SilentRejectCount: 20},
		{CampaignID: "c1", FraudReason: "datacenter_ip", EventCount: 50, SilentRejectCount: 5},
	}
	got := aggregateCustomerFraudByType(rows, fraudCategoryInvalidDevice)
	require.Len(t, got, 1)
	assert.Equal(t, fraudCategoryInvalidDevice, got[0].FraudCategory)
}
