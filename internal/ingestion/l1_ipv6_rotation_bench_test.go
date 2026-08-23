package ingestion

import (
	"testing"

	"github.com/google/uuid"
)

var ipv6RotationBenchSink bool

func BenchmarkIPv6Rotation_observe(b *testing.B) {
	table := NewIPv6RotationTable()
	table.SetMode("live")
	table.SetPolicy(uint64(defaultIPv6RotationWindow), 64)
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	campaignHash := crc32Castagnoli(&cid)
	v6Hi := uint64(0x20010db885a30000)
	now := monotonicNano()

	b.ReportAllocs()
	b.ResetTimer()
	var live bool
	for i := 0; i < b.N; i++ {
		live, _ = table.observe(campaignHash, v6Hi, uint64(i&63), now)
	}
	ipv6RotationBenchSink = ipv6RotationBenchSink || live
}

func BenchmarkIPv6Rotation_ClickHook(b *testing.B) {
	table := NewIPv6RotationTable()
	table.SetMode("shadow")
	table.SetPolicy(uint64(defaultIPv6RotationWindow), defaultIPv6RotationThresh)
	h := &AdsPacketHandler{
		registry: stubCampaignRegistry{
			camp: &cidrBenchCampaign,
			ok:   true,
		},
		ipv6RotationTable:   table,
		ipv6RotationMetrics: newL1IPv6RotationMetrics(),
	}
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	ips := []string{
		"2001:db8:85a3::1", "2001:db8:85a3::2", "2001:db8:85a3::3",
		"2001:db8:85a3::4", "2001:db8:85a3::5", "2001:db8:85a3::6",
	}
	parsed := &clickQueryParsed{ok: true, campaignID: cid}
	now := monotonicNano()

	b.ReportAllocs()
	b.ResetTimer()
	var block bool
	for i := 0; i < b.N; i++ {
		block = h.l1IPv6RotationObserve(ips[i%len(ips)], cid, parsed, now)
	}
	ipv6RotationBenchSink = ipv6RotationBenchSink || block
}
