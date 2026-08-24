package controlplane

import (
	"fmt"
	"testing"
)

func makeTrafficRows(n int) []TrafficSourceRowDTO {
	rows := make([]TrafficSourceRowDTO, n)
	for i := range rows {
		rows[i] = TrafficSourceRowDTO{
			Channel:    fmt.Sprintf("ch-%d", i),
			SpendMicro: int64((i * 17) % n * 1000),
			Clicks:     int64((i * 13) % n),
		}
	}
	return rows
}

func BenchmarkReportSort_Traffic(b *testing.B) {
	src := makeTrafficRows(500)
	for b.Loop() {
		rows := append([]TrafficSourceRowDTO(nil), src...)
		sortTrafficRows(rows)
	}
}

func BenchmarkReportSort_Geo(b *testing.B) {
	src := make([]GeoROIRowDTO, 500)
	for i := range src {
		src[i] = GeoROIRowDTO{
			Country:    fmt.Sprintf("US-%d", i),
			SpendMicro: int64((i * 19) % 500 * 1000),
			IVTEvents:  int64((i * 11) % 500),
		}
	}
	for b.Loop() {
		rows := append([]GeoROIRowDTO(nil), src...)
		sortGeoROIRows(rows)
	}
}
