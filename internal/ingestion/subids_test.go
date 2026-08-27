package ingestion

import "testing"

func TestSubKeyIndex(t *testing.T) {
	tests := []struct {
		key string
		idx int
		ok  bool
	}{
		{"sub1", 1, true},
		{"sub9", 9, true},
		{"sub10", 10, true},
		{"sub30", 30, true},
		{"sub0", 0, false},
		{"sub31", 0, false},
		{"subs1", 0, false},
		{"xsub1", 0, false},
	}
	for _, tt := range tests {
		idx, ok := subKeyIndex([]byte(tt.key))
		if ok != tt.ok || idx != tt.idx {
			t.Errorf("subKeyIndex(%q) = (%d, %v), want (%d, %v)", tt.key, idx, ok, tt.idx, tt.ok)
		}
	}
}

func TestAppendAttributionPayload(t *testing.T) {
	var subs SubIDSlots
	subs[0] = "fb"
	subs[9] = "ten"
	out := appendAttributionPayload(nil, nil, subs, "fbc", "gc", "ttc", "ms1", "tb1", "ob1", "evt-1", "tx-1")
	want := `{"sub1":"fb","sub10":"ten","fbclid":"fbc","gclid":"gc","ttclid":"ttc","msclkid":"ms1","tblci":"tb1","ob_click_id":"ob1","event_id":"evt-1","tx_id":"tx-1"}`
	if string(out) != want {
		t.Fatalf("got %s want %s", out, want)
	}

	nested := appendAttributionPayload(nil, []byte(`{"fault":"1"}`), subs, "", "", "", "", "", "", "", "")
	wantNested := `{"fault":"1","sub1":"fb","sub10":"ten"}`
	if string(nested) != wantNested {
		t.Fatalf("nested got %s want %s", nested, wantNested)
	}
}

func TestParseTrackRequestJSON_SubIDs(t *testing.T) {
	data := []byte(`{"campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","type":"conversion","sub1":"a","sub30":"z","fbclid":"f","gclid":"g","ttclid":"t","msclkid":"m","tblci":"tb","ob_click_id":"ob","event_id":"evt-9","tx_id":"tx-9"}`)
	var req TrackRequest
	if err := ParseTrackRequestJSONOpt(&req, data); err != nil {
		t.Fatal(err)
	}
	if req.subs[0] != "a" || req.subs[29] != "z" {
		t.Fatalf("subs=%v %v", req.subs[0], req.subs[29])
	}
	if req.fbclid != "f" || req.gclid != "g" || req.ttclid != "t" {
		t.Fatalf("attribution ids missing")
	}
	if req.msclkid != "m" || req.tblci != "tb" || req.obClickID != "ob" {
		t.Fatalf("network click ids missing")
	}
	if req.eventID != "evt-9" || req.txID != "tx-9" {
		t.Fatalf("event_id/tx_id missing: %q %q", req.eventID, req.txID)
	}
}

func TestTrackCORS(t *testing.T) {
	cors := newTrackCORS([]string{"https://lp.example", "*"})
	if !cors.match("https://lp.example") {
		t.Fatal("expected explicit origin")
	}
	if !cors.match("https://other.example") {
		t.Fatal("expected wildcard reflect")
	}
	resp := buildTrackCORSPreflight("https://lp.example", cors)
	if len(resp) < 50 {
		t.Fatalf("preflight response too short: %q", resp)
	}
}
