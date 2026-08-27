package postback

import "testing"

func TestResolveEventID_prefersEventID(t *testing.T) {
	pb := &PostbackPayload{
		EventID: "evt-1",
		TxID:    "tx-1",
		ClickID: "clk-1",
	}
	if got := ResolveEventID(pb); got != "evt-1" {
		t.Fatalf("ResolveEventID() = %q want evt-1", got)
	}
}

func TestResolveEventID_fallsBackToTxThenClick(t *testing.T) {
	if got := ResolveEventID(&PostbackPayload{TxID: "tx-9", ClickID: "clk-9"}); got != "tx-9" {
		t.Fatalf("tx fallback = %q", got)
	}
	if got := ResolveEventID(&PostbackPayload{ClickID: "clk-9"}); got != "clk-9" {
		t.Fatalf("click fallback = %q", got)
	}
}

func TestPostbackClickIDWarnings_meta(t *testing.T) {
	w := PostbackClickIDWarnings("facebook", &PostbackPayload{})
	if len(w) != 1 || w[0] == "" {
		t.Fatalf("warnings = %#v", w)
	}
	if len(PostbackClickIDWarnings("facebook", &PostbackPayload{FBCLID: "fb1"})) != 0 {
		t.Fatal("expected no warnings when fbclid set")
	}
}
