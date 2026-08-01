package rtb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuction_bcatBlocksCategory(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	geo := uint32(7)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 500, DeviceMask: 1, CategoryMask: 4, GeoHashVal: geo, Weight: 1, Budget: 10_000,
	}})
	req := &BidRequest{
		GeoHash: geo, DeviceType: 1, CategoryMask: 1, MinBid: 50, BlockedCatMask: 4,
	}
	res, reason := reg.RunAuction(req)
	assert.False(t, reason.OK())
	assert.Equal(t, NoBidNoCandidates, reason)
	assert.Equal(t, CampaignID(0), res.CampaignID)
}

func TestAuction_bcatAllowsOtherCategories(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	geo := uint32(7)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 500, DeviceMask: 1, CategoryMask: 1, GeoHashVal: geo, Weight: 1, Budget: 10_000,
	}})
	req := &BidRequest{
		GeoHash: geo, DeviceType: 1, CategoryMask: 1, MinBid: 50, BlockedCatMask: 4,
	}
	_, reason := reg.RunAuction(req)
	require.True(t, reason.OK())
}
