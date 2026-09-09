package domain

import (
	"hash/crc32"
	"testing"
)

func TestGeoHashFromCountry_matchesChecksumIEEE(t *testing.T) {
	cases := []string{"", "US", "DE", "GB", "RU", "XX"}
	for _, country := range cases {
		want := crc32.ChecksumIEEE([]byte(country))
		got := GeoHashFromCountry(country)
		if got != want {
			t.Fatalf("country=%q got=%d want=%d", country, got, want)
		}
	}
}

func TestGeoHashFromCountry_zeroAlloc(t *testing.T) {
	const country = "US"
	allocs := testing.AllocsPerRun(100, func() {
		_ = GeoHashFromCountry(country)
	})
	if allocs != 0 {
		t.Fatalf("GeoHashFromCountry allocs=%v want 0", allocs)
	}
}
