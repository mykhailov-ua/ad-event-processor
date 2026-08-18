package controlplane

import (
	"fmt"
	"testing"
)

func legacySortTrafficRows(rows []TrafficSourceRowDTO) {
	for i := range rows {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].SpendMicro > rows[i].SpendMicro ||
				(rows[j].SpendMicro == rows[i].SpendMicro && rows[j].Clicks > rows[i].Clicks) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

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

func BenchmarkReportSort_Traffic_Legacy(b *testing.B) {
	src := makeTrafficRows(500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := append([]TrafficSourceRowDTO(nil), src...)
		legacySortTrafficRows(rows)
	}
}

func BenchmarkReportSort_Traffic_Batched(b *testing.B) {
	src := makeTrafficRows(500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := append([]TrafficSourceRowDTO(nil), src...)
		sortTrafficRows(rows)
	}
}

func BenchmarkReportSort_Geo_Legacy(b *testing.B) {
	src := make([]GeoROIRowDTO, 500)
	for i := range src {
		src[i] = GeoROIRowDTO{
			Country:    fmt.Sprintf("US-%d", i),
			SpendMicro: int64((i * 19) % 500 * 1000),
			IVTEvents:  int64((i * 11) % 500),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := append([]GeoROIRowDTO(nil), src...)
		for a := range rows {
			for c := a + 1; c < len(rows); c++ {
				if rows[c].SpendMicro > rows[a].SpendMicro ||
					(rows[c].SpendMicro == rows[a].SpendMicro && rows[c].IVTEvents > rows[a].IVTEvents) {
					rows[a], rows[c] = rows[c], rows[a]
				}
			}
		}
	}
}

func BenchmarkReportSort_Geo_Batched(b *testing.B) {
	src := make([]GeoROIRowDTO, 500)
	for i := range src {
		src[i] = GeoROIRowDTO{
			Country:    fmt.Sprintf("US-%d", i),
			SpendMicro: int64((i * 19) % 500 * 1000),
			IVTEvents:  int64((i * 11) % 500),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := append([]GeoROIRowDTO(nil), src...)
		sortGeoROIRows(rows)
	}
}
