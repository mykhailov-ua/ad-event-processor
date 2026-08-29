package ingest

import (
	"math/rand"
	"net/netip"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

var (
	cidrBenchSinkBool  bool
	cidrBenchSinkUint8 uint8

	cidrBenchCampaign = domain.Campaign{CIDRBlockEnabled: true}
)

func benchCIDRTable(tb testing.TB, n int) (*CIDRTable, [][4]byte, [][16]byte) {
	tb.Helper()
	rng := rand.New(rand.NewSource(2026))
	var b cidrBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for i := range n {
		if i%4 == 3 {
			var a [16]byte
			rng.Read(a[:])
			a[0], a[1] = 0x26, 0x20
			b.Insert(&root6, a, uint8(16+rng.Intn(49)), uint8(i%int(CIDRFeedCount)))
			continue
		}
		var a [4]byte
		rng.Read(a[:])
		var key [16]byte
		copy(key[:4], a[:])
		b.Insert(&root4, key, uint8(8+rng.Intn(25)), uint8(i%int(CIDRFeedCount)))
	}
	table := NewCIDRTable()
	table.Publish(b.Snapshot(root4, root6, 1))

	probe4 := make([][4]byte, 64)
	probe6 := make([][16]byte, 64)
	for i := range probe4 {
		rng.Read(probe4[i][:])
		rng.Read(probe6[i][:])
		probe6[i][0], probe6[i][1] = 0x26, 0x20
	}
	return table, probe4, probe6
}

func BenchmarkCIDR_LPM_Lookup_IPv4(b *testing.B) {
	table, probe4, _ := benchCIDRTable(b, 50_000)
	b.ReportAllocs()
	var hit bool
	var feed uint8
	benchN := 0
	for b.Loop() {
		hit, feed = table.Match4(probe4[benchN&63])
		benchN++
	}
	cidrBenchSinkUint8 += feed
	cidrBenchSinkBool = cidrBenchSinkBool || hit
}

func BenchmarkCIDR_LPM_Lookup_IPv6(b *testing.B) {
	table, _, probe6 := benchCIDRTable(b, 50_000)
	b.ReportAllocs()
	var hit bool
	var feed uint8
	benchN := 0
	for b.Loop() {
		hit, feed = table.Match6(probe6[benchN&63])
		benchN++
	}
	cidrBenchSinkUint8 += feed
	cidrBenchSinkBool = cidrBenchSinkBool || hit
}

func BenchmarkCIDR_LPM_Lookup_ParseIP(b *testing.B) {
	table, _, _ := benchCIDRTable(b, 50_000)
	ips := []string{
		"54.230.17.9", "2001:4860:4860::8888", "203.0.113.7",
		"142.250.74.46", "2606:4700:4700::1111", "198.51.100.23",
	}
	b.ReportAllocs()
	var hit bool
	var feed uint8
	benchN := 0
	for b.Loop() {
		hit, feed = table.MatchIP(ips[benchN%len(ips)])
		benchN++
	}
	cidrBenchSinkUint8 += feed
	cidrBenchSinkBool = cidrBenchSinkBool || hit
}

func BenchmarkCIDR_MatchBranch_SafeView(b *testing.B) {
	table, probe4, _ := benchCIDRTable(b, 50_000)
	h := &AdsPacketHandler{
		registry: stubCampaignRegistry{
			camp: &cidrBenchCampaign,
			ok:   true,
		},
		cidrTable:   table,
		cidrMetrics: newCIDRBlockMetrics(),
	}
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	ipStrs := make([]string, len(probe4))
	for i := range probe4 {
		ipStrs[i] = netip.AddrFrom4(probe4[i]).String()
	}
	b.ReportAllocs()
	var hit bool
	benchN := 0
	for b.Loop() {
		hit, _ = h.cidrBlockShouldSafeView(ipStrs[benchN&63], cid)
		benchN++
	}
	cidrBenchSinkBool = cidrBenchSinkBool || hit
}
