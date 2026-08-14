package rtb

import (
	"testing"
)

func BenchmarkRunAuction_daypartGate(b *testing.B) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)
	n := 1000
	campaigns := make([]CampaignData, n)
	nowUnix := int64(1_700_000_000)

	for i := range n {
		campaigns[i] = CampaignData{
			ID:           CampaignID(uint64(i + 1)),
			Bid:          int64(100 + (i % 500)),
			DeviceMask:   uint8(1 << (i % 3)),
			CategoryMask: uint64(1 << (i % 8)),
			GeoHashVal:   uint32(i),
			Weight:       uint32(i),
			Budget:       1000000000,
			DaypartMask:  1 << 12,
		}
	}
	reg.UpdateCampaigns(campaigns)

	req := &BidRequest{
		DeviceType:   2,
		CategoryMask: 4,
		GeoHash:      2,
		MinBid:       150,
		NowUnix:      nowUnix,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.RunAuction(req)
	}
}

func BenchmarkRunAuction_freqCapGate(b *testing.B) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)
	prefixHash := HashBytes64([]byte("{abc}fcap:c:bench:u:"))
	userHash := HashBytes64([]byte("bench-user"))
	reg.SetFcapSnapshot(NewFcapSnapshot(map[uint64]uint32{
		FcapLookupKey(prefixHash, userHash): 0,
	}))

	n := 1000
	campaigns := make([]CampaignData, n)
	for i := range n {
		campaigns[i] = CampaignData{
			ID:             CampaignID(uint64(i + 1)),
			Bid:            int64(100 + (i % 500)),
			DeviceMask:     uint8(1 << (i % 3)),
			CategoryMask:   uint64(1 << (i % 8)),
			GeoHashVal:     uint32(i),
			Weight:         uint32(i),
			Budget:         1000000000,
			FreqLimit:      5,
			FcapPrefixHash: prefixHash,
		}
	}
	reg.UpdateCampaigns(campaigns)

	req := &BidRequest{
		DeviceType:   2,
		CategoryMask: 4,
		GeoHash:      2,
		MinBid:       150,
		FcapUserHash: userHash,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.RunAuction(req)
	}
}
