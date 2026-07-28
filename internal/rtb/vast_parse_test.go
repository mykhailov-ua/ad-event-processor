package rtb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVASTXML_corpus(t *testing.T) {
	dir := filepath.Join("testdata", "vast")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		doc, err := ParseVASTXML(raw)
		if err != nil {
			t.Fatalf("%s parse: %v", name, err)
		}
		wire, err := MarshalVASTDocument(doc)
		if err != nil {
			t.Fatalf("%s marshal: %v", name, err)
		}
		round, err := UnmarshalVASTDocument(wire)
		if err != nil {
			t.Fatalf("%s unmarshal vtproto: %v", name, err)
		}
		if VASTDurationSec(round) == 0 {
			t.Fatalf("%s: expected positive duration", name)
		}
		if mime := VASTMediaMIME(round); mime == "" {
			t.Fatalf("%s: expected media mime", name)
		}
	}
}

func TestRunAuction_multiCreativeWinner(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	campaigns := []CampaignData{{
		ID:           42,
		Bid:          100,
		DeviceMask:   1,
		CategoryMask: 1,
		GeoHashVal:   7,
		Weight:       1,
		Budget:       1_000_000,
	}}
	raw, err := os.ReadFile(filepath.Join("testdata", "vast", "inline_30s.xml"))
	if err != nil {
		t.Fatal(err)
	}
	reg.UpdateCreatives([]CreativeData{
		{ID: 1001, CampaignID: 42, Bid: 500, Weight: 1, MediaType: MediaTypeVideo, VASTXML: raw},
		{ID: 1002, CampaignID: 42, Bid: 300, Weight: 1, MediaType: MediaTypeVideo, VASTXML: raw},
	})
	reg.UpdateCampaigns(campaigns)

	res, reason := reg.RunAuction(&BidRequest{
		DeviceType:    1,
		CategoryMask:  1,
		GeoHash:       7,
		MinBid:        50,
		MediaTypeMask: uint8(MediaTypeVideo),
	})
	if !reason.OK() {
		t.Fatalf("expected ok, got %s", reason)
	}
	if res.CampaignID != 42 {
		t.Fatalf("campaign: got %d want 42", res.CampaignID)
	}
	if res.CreativeID != 1001 {
		t.Fatalf("creative: got %d want 1001 (higher bid)", res.CreativeID)
	}
}

func TestRunAuction_videoDurationFilter(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	campaigns := []CampaignData{{
		ID:           7,
		Bid:          100,
		DeviceMask:   1,
		CategoryMask: 1,
		GeoHashVal:   7,
		Budget:       1_000_000,
	}}
	reg.UpdateCreatives([]CreativeData{
		{ID: 2001, CampaignID: 7, Bid: 400, MediaType: MediaTypeVideo, DurationSec: 15},
		{ID: 2002, CampaignID: 7, Bid: 600, MediaType: MediaTypeVideo, DurationSec: 45},
	})
	reg.UpdateCampaigns(campaigns)

	res, reason := reg.RunAuction(&BidRequest{
		DeviceType:     1,
		CategoryMask:   1,
		GeoHash:        7,
		MinBid:         50,
		MediaTypeMask:  uint8(MediaTypeVideo),
		MaxDurationSec: 20,
	})
	if !reason.OK() {
		t.Fatalf("expected ok, got %s", reason)
	}
	if res.CreativeID != 2001 {
		t.Fatalf("creative: got %d want 2001 (45s filtered)", res.CreativeID)
	}
}
