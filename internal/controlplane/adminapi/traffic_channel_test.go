package adminapi

import "testing"

func TestClassifyTrafficChannel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sub1, sub2, gclid, fbclid, ttclid, want string
	}{
		{"", "", "G1", "", "", "paid_search"},
		{"", "", "", "FB1", "", "paid_social"},
		{"", "", "", "", "TT1", "paid_social"},
		{"organic", "", "", "", "", "organic"},
		{"", "email", "", "", "", "email"},
		{"affiliate", "", "", "", "", "affiliate"},
		{"fb", "", "", "", "", "custom"},
		{"", "", "", "", "", "direct"},
	}
	for _, tc := range cases {
		got := classifyTrafficChannel(tc.sub1, tc.sub2, tc.gclid, tc.fbclid, tc.ttclid)
		if got != tc.want {
			t.Fatalf("classifyTrafficChannel(%q,%q,%q,%q,%q) = %q, want %q",
				tc.sub1, tc.sub2, tc.gclid, tc.fbclid, tc.ttclid, got, tc.want)
		}
	}
}
