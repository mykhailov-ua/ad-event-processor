package coldpath

import "testing"

func TestFormatMicroDisplay_groupedASCII(t *testing.T) {
	cases := map[int64]string{
		0:          "0",
		42:         "42",
		1000:       "1,000",
		1234567:    "1,234,567",
		-1234567:   "-1,234,567",
		1500000000: "1,500,000,000",
	}
	for micro, want := range cases {
		if got := FormatMicroDisplay(micro); got != want {
			t.Fatalf("FormatMicroDisplay(%d) = %q, want %q", micro, got, want)
		}
	}
}

func TestFormatCountDisplay_matchesMicroDisplay(t *testing.T) {
	const n int64 = 9876543210
	if got := FormatCountDisplay(n); got != FormatMicroDisplay(n) {
		t.Fatalf("FormatCountDisplay() = %q, want %q", got, FormatMicroDisplay(n))
	}
}
