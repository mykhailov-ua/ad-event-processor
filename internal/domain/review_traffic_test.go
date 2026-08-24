package domain

import "testing"

func TestParseReviewTrafficAction(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want ReviewTrafficAction
	}{
		{"safe_page", ReviewTrafficActionSafePage},
		{"block", ReviewTrafficActionBlock},
		{"passthrough", ReviewTrafficActionPassthrough},
		{"", ReviewTrafficActionSafePage},
		{"CLOAK", ReviewTrafficActionSafePage},
	}
	for _, tc := range cases {
		got := ParseReviewTrafficAction(tc.in)
		if got != tc.want {
			t.Fatalf("ParseReviewTrafficAction(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
