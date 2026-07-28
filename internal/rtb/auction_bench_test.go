package rtb

import (
	"os"
	"path/filepath"
	"testing"
)

// Tracks hot-path auction latency with campaigns spread across geo shards.
func BenchmarkAuction(b *testing.B) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)
	n := 1000
	campaigns := make([]CampaignData, n)

	for i := 0; i < n; i++ {
		deviceMask := uint8(1 << (i % 3))
		campaigns[i] = CampaignData{
			ID:           CampaignID(uint64(i + 1)),
			Bid:          int64(100 + (i % 500)),
			DeviceMask:   deviceMask,
			CategoryMask: uint64(1 << (i % 8)),
			GeoHashVal:   uint32(i),
			Weight:       uint32(i),
			Budget:       1000000000,
		}
	}
	reg.UpdateCampaigns(campaigns)

	req := &BidRequest{
		DeviceType:   2,
		CategoryMask: 4,
		GeoHash:      2,
		MinBid:       150,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.RunAuction(req)
	}
}

// Tracks worst-case scan cost when many campaigns share one geo shard.
func BenchmarkAuction_highDensity(b *testing.B) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)
	n := 1000
	campaigns := make([]CampaignData, n)

	for i := 0; i < n; i++ {
		campaigns[i] = CampaignData{
			ID:           CampaignID(uint64(i + 1)),
			Bid:          int64(100 + i),
			DeviceMask:   1,
			CategoryMask: 1,
			GeoHashVal:   5,
			Weight:       uint32(i),
			Budget:       1000000000,
		}
	}
	reg.UpdateCampaigns(campaigns)

	req := &BidRequest{
		DeviceType:   1,
		CategoryMask: 1,
		GeoHash:      5,
		MinBid:       50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.RunAuction(req)
	}
}

// Tracks multi-creative video auction latency with creative-level bucket materialization.
func BenchmarkRunAuction_MultiCreative(b *testing.B) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)
	n := 200
	campaigns := make([]CampaignData, n)
	creatives := make([]CreativeData, 0, n*3)
	vastWire := mustBenchVASTWire(b)

	for i := 0; i < n; i++ {
		cid := CampaignID(uint64(i + 1))
		deviceMask := uint8(1 << (i % 3))
		campaigns[i] = CampaignData{
			ID:           cid,
			Bid:          int64(100 + (i % 500)),
			DeviceMask:   deviceMask,
			CategoryMask: uint64(1 << (i % 8)),
			GeoHashVal:   uint32(i),
			Weight:       uint32(i),
			Budget:       1000000000,
		}
		for c := 0; c < 3; c++ {
			creatives = append(creatives, CreativeData{
				ID:          CreativeID(uint64(i*10 + c + 1)),
				CampaignID:  cid,
				Bid:         int64(120 + (i % 400) + c*10),
				Weight:      uint32(c + 1),
				MediaType:   MediaTypeVideo,
				VASTWire:    vastWire,
				DurationSec: 30,
			})
		}
	}
	reg.UpdateCreatives(creatives)
	reg.UpdateCampaigns(campaigns)

	req := &BidRequest{
		DeviceType:    2,
		CategoryMask:  4,
		GeoHash:       2,
		MinBid:        150,
		MediaTypeMask: uint8(MediaTypeVideo),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.RunAuction(req)
	}
}

func mustBenchVASTWire(b *testing.B) []byte {
	b.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "vast", "inline_30s.xml"))
	if err != nil {
		b.Fatal(err)
	}
	doc, err := ParseVASTXML(raw)
	if err != nil {
		b.Fatal(err)
	}
	wire, err := MarshalVASTDocument(doc)
	if err != nil {
		b.Fatal(err)
	}
	return wire
}
