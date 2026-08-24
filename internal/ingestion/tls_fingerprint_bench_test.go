package ingestion

import (
	"hash/crc32"
	"testing"

	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

var tlsFingerprintBenchSink bool

func benchTLSFingerprintTable(tb testing.TB, n int) (*TLSFingerprintTable, [][]byte) {
	tb.Helper()
	lines := make([]byte, 0, n*32)
	for i := 0; i < n; i++ {
		lines = append(lines, "ja3:771,4865-"...)
		lines = appendInt64(lines, int64(i))
		lines = append(lines, '\n')
	}
	ja3, ja4 := parseTLSFingerprintFeed(lines)
	table := NewTLSFingerprintTable()
	table.Publish(buildTLSFingerprintSnapshot(ja3, ja4, nil, nil, 1))

	probes := make([][]byte, 64)
	for i := range probes {
		probes[i] = appendInt64(nil, int64(i))
	}
	return table, probes
}

func BenchmarkTLS_Fingerprint_Lookup(b *testing.B) {
	table, probes := benchTLSFingerprintTable(b, 10_000)
	ja3 := []byte("771,4865-42")
	b.ReportAllocs()
	var hit bool
	benchN := 0
	for b.Loop() {
		hit = table.MatchJA3(probes[benchN&63]) || table.MatchJA3(ja3)
		benchN++
	}
	tlsFingerprintBenchSink = tlsFingerprintBenchSink || hit
}

func BenchmarkTLS_Fingerprint_MatchBranch_SafeView(b *testing.B) {
	table, probes := benchTLSFingerprintTable(b, 50_000)
	h := &AdsPacketHandler{
		tlsFingerprintTable:   table,
		tlsFingerprintMetrics: newTLSFingerprintMetrics(),
		registry: stubCampaignRegistry{
			camp: &domain.Campaign{TLSFingerprintBlockEnabled: true},
			ok:   true,
		},
	}
	b.ReportAllocs()
	var hit bool
	benchN := 0
	for b.Loop() {
		hit, _ = h.tlsFingerprintShouldSafeView(probes[benchN&63], nil, uuidNil, "")
		benchN++
	}
	tlsFingerprintBenchSink = tlsFingerprintBenchSink || hit
}

func BenchmarkTLS_Fingerprint_AllowlistBranch(b *testing.B) {
	ja3 := []byte("771,4865-4866,0-23,29-23-24,0")
	h := crc32.ChecksumIEEE(ja3)
	table := NewTLSFingerprintTable()
	table.Publish(buildTLSFingerprintSnapshot([]uint32{h}, nil, []uint32{h}, nil, 1))
	b.ReportAllocs()
	var hit bool
	for b.Loop() {
		hit = table.shouldBlockJA3(ja3)
	}
	tlsFingerprintBenchSink = tlsFingerprintBenchSink || hit
}

var uuidNil = uuid.MustParse("00000000-0000-4000-8000-000000000001")
