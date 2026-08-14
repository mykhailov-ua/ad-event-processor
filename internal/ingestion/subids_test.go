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
	out := appendAttributionPayload(nil, nil, subs, "fbc", "gc", "ttc")
	want := `{"sub1":"fb","sub10":"ten","fbclid":"fbc","gclid":"gc","ttclid":"ttc"}`
	if string(out) != want {
		t.Fatalf("got %s want %s", out, want)
	}

	nested := appendAttributionPayload(nil, []byte(`{"fault":"1"}`), subs, "", "", "")
	wantNested := `{"fault":"1","sub1":"fb","sub10":"ten"}`
	if string(nested) != wantNested {
		t.Fatalf("nested got %s want %s", nested, wantNested)
	}
}

func TestParseTrackRequestJSON_SubIDs(t *testing.T) {
	data := []byte(`{"campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","type":"conversion","sub1":"a","sub30":"z","fbclid":"f","gclid":"g","ttclid":"t"}`)
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
