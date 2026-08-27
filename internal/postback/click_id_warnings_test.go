package postback

import (
	"strings"
	"testing"
)

func TestPostbackClickIDWarnings_allProviders(t *testing.T) {
	cases := []struct {
		provider string
		payload  *PostbackPayload
		wantSub  string
	}{
		{"facebook", &PostbackPayload{}, "missing_fbclid"},
		{"facebook", &PostbackPayload{FBCLID: "fb-1"}, ""},
		{"google", &PostbackPayload{}, "missing_gclid"},
		{"google", &PostbackPayload{GCLID: "g-1"}, ""},
		{"tiktok", &PostbackPayload{}, "missing_ttclid"},
		{"tiktok", &PostbackPayload{TTCLID: "tt-1"}, ""},
		{"taboola", &PostbackPayload{}, "missing_tblci"},
		{"taboola", &PostbackPayload{TBLCI: "tb-1"}, ""},
		{"outbrain", &PostbackPayload{}, "missing_ob_click_id"},
		{"outbrain", &PostbackPayload{OBClickID: "ob-1"}, ""},
		{"microsoft_ads", &PostbackPayload{}, "missing_msclkid"},
		{"microsoft_ads", &PostbackPayload{MSCLKID: "ms-1"}, ""},
		{"webhook", &PostbackPayload{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			got := PostbackClickIDWarnings(tc.provider, tc.payload)
			if tc.wantSub == "" {
				if len(got) != 0 {
					t.Fatalf("warnings = %#v", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.wantSub) {
				t.Fatalf("warnings = %#v want substring %q", got, tc.wantSub)
			}
		})
	}
}
