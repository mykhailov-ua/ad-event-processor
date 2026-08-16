package postback

import (
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

func TestEventTypeMatches(t *testing.T) {
	if !eventTypeMatches("conversion", "conversion") {
		t.Fatal("exact match")
	}
	if !eventTypeMatches("Conversion", "conversion") {
		t.Fatal("case insensitive")
	}
	if !eventTypeMatches("conversion", "") {
		t.Fatal("empty target defaults to conversion")
	}
	if eventTypeMatches("click", "conversion") {
		t.Fatal("mismatch")
	}
}

func TestMergeEventPayloadInto(t *testing.T) {
	var pb PostbackPayload
	mergeEventPayloadInto(&pb, []byte(`{
		"sub1":"src",
		"sub30":"deep",
		"fbclid":"fb1",
		"gclid":"gc1",
		"ttclid":"tt1",
		"email":"a@b.c",
		"payout_micro": 2500000
	}`))
	if pb.SubID1 != "src" || pb.subSlots[29] != "deep" {
		t.Fatalf("subs: %q %q", pb.SubID1, pb.subSlots[29])
	}
	if pb.FBCLID != "fb1" || pb.GCLID != "gc1" || pb.TTCLID != "tt1" {
		t.Fatalf("click ids")
	}
	if pb.Email != "a@b.c" {
		t.Fatalf("email %q", pb.Email)
	}
	if pb.PayoutMicro != 2_500_000 {
		t.Fatalf("payout %d", pb.PayoutMicro)
	}
}

func TestBuildPostbackPayloadFromEvent(t *testing.T) {
	cid := uuid.New()
	cust := uuid.New()
	evt := &domain.Event{
		ClickID:            "clk-1",
		CampaignID:         cid,
		Type:               "conversion",
		ClearingPriceMicro: 1_000_000,
		Payload:            []byte(`{"gclid":"G-9","sub2":"x"}`),
	}
	pb := buildPostbackPayloadFromEvent(evt, cust)
	if pb.CustomerID != cust || pb.CampaignID != cid || pb.ClickID != "clk-1" {
		t.Fatalf("ids %+v", pb)
	}
	if pb.GCLID != "G-9" || pb.subSlots[1] != "x" {
		t.Fatalf("attribution %+v", pb)
	}
	if pb.EventSourceURL == "" || !strings.Contains(pb.EventSourceURL, "click_id=clk-1") {
		t.Fatalf("event_source_url %q", pb.EventSourceURL)
	}
	if pb.PayoutMicro != 1_000_000 || pb.TxID != "clk-1" {
		t.Fatalf("payout/tx %d %q", pb.PayoutMicro, pb.TxID)
	}
}
