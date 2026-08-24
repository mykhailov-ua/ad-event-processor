package ingestion

import (
	"math/rand"
	"net/netip"
	"testing"

	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

var proxyVPNBenchSink struct {
	match    bool
	connType uint8
	asn      uint32
}

func benchProxyVPNTable(tb testing.TB, n int) (*ProxyVPNTable, [][4]byte) {
	tb.Helper()
	rng := rand.New(rand.NewSource(2026))
	var b proxyVPNBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for i := 0; i < n; i++ {
		var a [4]byte
		rng.Read(a[:])
		bits := 8 + rng.Intn(24)
		prefix := netip.PrefixFrom(netip.AddrFrom4(a), bits).Masked()
		b.addPrefix(prefix, ProxyVPNConnVPN|ProxyVPNConnHosting, uint32(64512+i), &root4, &root6)
	}
	table := NewProxyVPNTable()
	table.Publish(b.snapshot(root4, root6, 1))

	probes := make([][4]byte, 64)
	for i := range probes {
		rng.Read(probes[i][:])
	}
	return table, probes
}

func BenchmarkProxyVPN_Lookup(b *testing.B) {
	table, probes := benchProxyVPNTable(b, 10_000)
	b.ReportAllocs()
	b.ResetTimer()
	var match bool
	var connType uint8
	var asn uint32
	for i := 0; i < b.N; i++ {
		match, connType, asn = table.Lookup4(probes[i&63])
	}
	proxyVPNBenchSink.match = proxyVPNBenchSink.match || match
	proxyVPNBenchSink.connType += connType
	proxyVPNBenchSink.asn += asn
}

func BenchmarkProxyVPN_MatchBranch_SafeView(b *testing.B) {
	table, probes := benchProxyVPNTable(b, 50_000)
	h := &AdsPacketHandler{
		registry: stubCampaignRegistry{
			camp: &domain.Campaign{L15ProxyVPNBlockEnabled: true},
			ok:   true,
		},
		proxyVPNTable:      table,
		l15ProxyVPNMetrics: newL15ProxyVPNMetrics(),
	}
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	ipStrs := make([]string, len(probes))
	for i := range probes {
		ipStrs[i] = netip.AddrFrom4(probes[i]).String()
	}
	b.ReportAllocs()
	b.ResetTimer()
	var hit bool
	for i := 0; i < b.N; i++ {
		hit, _ = h.l15ProxyVPNShouldSafeView(ipStrs[i&63], cid)
	}
	proxyVPNBenchSink.match = proxyVPNBenchSink.match || hit
}

func BenchmarkProxyVPN_Extended_Lookup(b *testing.B) {
	table, probes := benchProxyVPNTable(b, 10_000)
	b.ReportAllocs()
	b.ResetTimer()
	var match bool
	var connType uint8
	for i := 0; i < b.N; i++ {
		match, connType, _ = table.Lookup4(probes[i&63])
		_ = connTypePolicyBlocks(domain.ConnTypeMobileOnly, match, connType)
	}
	proxyVPNBenchSink.match = proxyVPNBenchSink.match || match
}
