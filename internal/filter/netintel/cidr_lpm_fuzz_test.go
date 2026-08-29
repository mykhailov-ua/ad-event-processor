package netintel

import (
	"math/rand"
	"net/netip"
	"strings"
	"testing"
)

func FuzzCIDRParse(f *testing.F) {
	seeds := []string{
		"10.0.0.0/8", "192.168.1.1", "2001:db8::/32", "::1", "54.0.0.0/8",
		"10.0.0.0/33", "999.1.1.1/8", "10.0.0.0/", "/8", "", " ", "#comment",
		"10.0.0.0/8 extra", "0.0.0.0/0", "::/0", "fe80::1%eth0",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, err := cidrParseEntry(s)
		if err != nil {
			return
		}
		if !p.IsValid() {
			t.Fatalf("accepted %q -> invalid prefix %v", s, p)
		}
		if strings.Contains(s, "/") {
			if _, perr := netip.ParsePrefix(s); perr != nil {
				t.Fatalf("accepted %q as host but it has a slash", s)
			}
		}
	})
}

func FuzzCIDRMatch(f *testing.F) {
	table, prefs := buildTestTable(f,
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"54.0.0.0/8", "54.128.0.0/10", "54.128.64.0/24",
		"142.250.0.0/15", "2001:db8::/32", "2606:4700::/32",
		"0.0.0.0/0",
	)
	f.Add("10.1.2.3")
	f.Add("54.128.64.9")
	f.Add("2001:db8::1")
	f.Add("not-an-ip")
	f.Add("")
	f.Fuzz(func(t *testing.T, ip string) {
		got, _ := table.MatchIP(ip)
		a, err := netip.ParseAddr(ip)
		if err != nil {
			if got {
				t.Fatalf("MatchIP(%q) matched but netip rejects it", ip)
			}
			return
		}
		want := false
		for _, p := range prefs {
			if p.Contains(a) {
				want = true
				break
			}
		}
		if got != want {
			t.Fatalf("MatchIP(%q) = %v, oracle %v", ip, got, want)
		}
	})
}

func FuzzCIDRBuild(f *testing.F) {
	f.Add(uint64(1), uint64(2))
	f.Add(uint64(0xdeadbeef), uint64(42))
	f.Fuzz(func(t *testing.T, seedA, seedB uint64) {
		rng := rand.New(rand.NewSource(int64(seedA ^ seedB<<1)))
		var b cidrBuilder
		root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
		var prefs []netip.Prefix
		n := 1 + rng.Intn(64)
		for i := range n {
			var a4 [4]byte
			rng.Read(a4[:])
			p := netip.PrefixFrom(netip.AddrFrom4(a4), rng.Intn(33)).Masked()
			prefs = append(prefs, p)
			b.addPrefix(p, CIDRFeedOther, &root4, &root6)
		}
		table := NewCIDRTable()
		table.Publish(b.snapshot(root4, root6, 1))
		for i := range 200 {
			var a4 [4]byte
			rng.Read(a4[:])
			ip := netip.AddrFrom4(a4)
			got, _ := table.Match4(a4)
			want := false
			for _, p := range prefs {
				if p.Contains(ip) {
					want = true
					break
				}
			}
			if got != want {
				t.Fatalf("Match4(%v) = %v, oracle %v (prefixes=%v)", ip, got, want, prefs)
			}
		}
	})
}
