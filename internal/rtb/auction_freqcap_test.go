package rtb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFcapLookupKey_stable(t *testing.T) {
	prefix := HashBytes64([]byte("{abc}fcap:c:abc:u:"))
	user := HashBytes64([]byte("user-1"))
	k1 := FcapLookupKey(prefix, user)
	k2 := FcapLookupKey(prefix, user)
	assert.Equal(t, k1, k2)
	assert.NotZero(t, k1)
}

func TestFcapSnapshot_failOpenMissingKey(t *testing.T) {
	snap := NewFcapSnapshot(map[uint64]uint32{123: 5})
	count, ok := snap.FcapCount(999, 888)
	assert.False(t, ok)
	assert.Zero(t, count)
}

func TestFreqCapExceeded_table(t *testing.T) {
	tests := []struct {
		limit uint32
		count uint32
		want  bool
	}{
		{limit: 0, count: 10, want: false},
		{limit: 3, count: 2, want: false},
		{limit: 3, count: 3, want: true},
		{limit: 3, count: 5, want: true},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, FreqCapExceeded(tc.limit, tc.count))
	}
}

func TestAuction_freqCapReject(t *testing.T) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)

	prefix := "{abc}fcap:c:abc:u:"
	prefixHash := HashBytes64([]byte(prefix))
	userID := "bench-user"
	userHash := HashBytes64([]byte(userID))

	reg.SetFcapSnapshot(NewFcapSnapshot(map[uint64]uint32{
		FcapLookupKey(prefixHash, userHash): 5,
	}))

	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: 200, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7,
		Weight: 1, Budget: 5000, FreqLimit: 3, FcapPrefixHash: prefixHash,
	}})

	_, reason := reg.RunAuction(&BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: 50,
		FcapUserHash: userHash,
	})
	assert.Equal(t, NoBidFreqCapExceeded, reason)
}

func TestAuction_freqCapOK(t *testing.T) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)

	prefix := "{abc}fcap:c:abc:u:"
	prefixHash := HashBytes64([]byte(prefix))
	userHash := HashBytes64([]byte("ok-user"))

	reg.SetFcapSnapshot(NewFcapSnapshot(map[uint64]uint32{
		FcapLookupKey(prefixHash, userHash): 1,
	}))

	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: 200, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7,
		Weight: 1, Budget: 5000, FreqLimit: 3, FcapPrefixHash: prefixHash,
	}})

	res, reason := reg.RunAuction(&BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: 50,
		FcapUserHash: userHash,
	})
	require.Equal(t, NoBidNone, reason)
	assert.Equal(t, CampaignID(1), res.CampaignID)
}

func TestAuction_freqCapMissingSnapshotFailOpen(t *testing.T) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)

	prefixHash := HashBytes64([]byte("{abc}fcap:c:abc:u:"))
	userHash := HashBytes64([]byte("ghost-user"))

	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: 200, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7,
		Weight: 1, Budget: 5000, FreqLimit: 1, FcapPrefixHash: prefixHash,
	}})

	res, reason := reg.RunAuction(&BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: 50,
		FcapUserHash: userHash,
	})
	require.Equal(t, NoBidNone, reason)
	assert.Equal(t, CampaignID(1), res.CampaignID)
}
