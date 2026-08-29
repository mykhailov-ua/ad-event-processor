package netintel

import (
	"math/rand"
	"net/netip"
	"testing"
)

func buildTestTable(t testing.TB, cidrs ...string) (*CIDRTable, []netip.Prefix) {
	t.Helper()
	var b cidrBuilder
	var prefs []netip.Prefix
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for _, s := range cidrs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			t.Fatalf("bad test prefix %q: %v", s, err)
		}
		p = p.Masked()
		prefs = append(prefs, p)
		if p.Addr().Is4() {
			a4 := p.Addr().As4()
			var key [16]byte
			copy(key[:4], a4[:])
			b.insert(&root4, key, uint8(p.Bits()), CIDRFeedAWS)
		} else {
			b.insert(&root6, p.Addr().As16(), uint8(p.Bits()), CIDRFeedTor)
		}
	}
	snap := &cidrSnapshot{
		gen:   1,
		root4: root4,
		root6: root6,
		nodes: b.nodes,
		prefs: b.prefs,
	}
	table := NewCIDRTable()
	table.Publish(snap)
	return table, prefs
}

func oracleMatch(prefs []netip.Prefix, ip string) bool {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, p := range prefs {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func TestCIDRTable_LPM_Basic(t *testing.T) {
	table, _ := buildTestTable(t,
		"54.0.0.0/8",
		"10.0.0.0/8",
		"2001:db8::/32",
	)
	cases := []struct {
		ip    string
		match bool
	}{
		{"54.1.2.3", true},
		{"54.255.255.255", true},
		{"55.0.0.1", false},
		{"10.9.9.9", true},
		{"11.0.0.1", false},
		{"2001:db8::dead:beef", true},
		{"2001:db9::1", false},
		{"::ffff:54.0.0.1", true},
		{"8.8.8.8", false},
		{"not-an-ip", false},
		{"", false},
	}
	for _, tc := range cases {
		got, _ := table.MatchIP(tc.ip)
		if got != tc.match {
			t.Errorf("MatchIP(%q) = %v, want %v", tc.ip, got, tc.match)
		}
	}
}

func TestCIDRTable_NestedPrefixes(t *testing.T) {
	table, _ := buildTestTable(t,
		"10.0.0.0/8",
		"10.128.0.0/9",
		"10.0.0.0/16",
		"10.0.128.0/17",
	)
	cases := []struct {
		ip    string
		match bool
	}{
		{"10.0.0.1", true},
		{"10.0.200.1", true},
		{"10.0.64.1", true},
		{"10.200.1.1", true},
		{"10.64.0.1", true},
		{"11.0.0.1", false},
		{"9.255.255.255", false},
	}
	for _, tc := range cases {
		got, _ := table.MatchIP(tc.ip)
		if got != tc.match {
			t.Errorf("MatchIP(%q) = %v, want %v", tc.ip, got, tc.match)
		}
	}
}

func TestCIDRTable_InsertOrderShorterAfterLonger(t *testing.T) {
	table, _ := buildTestTable(t,
		"10.0.128.0/17",
		"10.0.0.0/16",
		"10.128.0.0/9",
		"10.0.0.0/8",
	)
	for _, ip := range []string{"10.0.64.1", "10.200.3.4", "10.0.200.9", "10.255.255.255"} {
		got, _ := table.MatchIP(ip)
		if !got {
			t.Errorf("MatchIP(%q) = false, want true (all inside 10.0.0.0/8)", ip)
		}
	}
	for _, ip := range []string{"11.0.0.1", "9.0.0.1"} {
		got, _ := table.MatchIP(ip)
		if got {
			t.Errorf("MatchIP(%q) = true, want false", ip)
		}
	}
}

func TestCIDRTable_HostBitsAndBoundaries(t *testing.T) {
	table, _ := buildTestTable(t,
		"192.168.0.0/24",
		"203.0.113.0/25",
		"0.0.0.0/0",
		"::/0",
	)
	for _, ip := range []string{"1.2.3.4", "255.255.255.255", "0.0.0.0", "2001:db8::1", "::1"} {
		got, _ := table.MatchIP(ip)
		if !got {
			t.Errorf("MatchIP(%q) = false, want true under /0", ip)
		}
	}
}

func TestCIDRTable_EmptyAndNil(t *testing.T) {
	table := NewCIDRTable()
	if got, _ := table.MatchIP("54.1.2.3"); got {
		t.Fatal("uninitialized table must fail open (no match)")
	}
	table.Publish(&cidrSnapshot{root4: cidrNoIndex, root6: cidrNoIndex, nodes: []CIDRNode{}})

	if table.Ready() != true {
		t.Fatal("published empty snapshot must report ready")
	}
}

func TestCIDRTable_OracleRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var cidrs []string
	for i := range 300 {
		var a [4]byte
		rng.Read(a[:])
		bits := 8 + rng.Intn(25)
		p := netip.PrefixFrom(netip.AddrFrom4(a), bits).Masked()
		cidrs = append(cidrs, p.String())
	}
	for i := range 100 {
		var a [16]byte
		rng.Read(a[:])
		bits := 32 + rng.Intn(97)
		p := netip.PrefixFrom(netip.AddrFrom16(a), bits).Masked()
		cidrs = append(cidrs, p.String())
	}
	table, prefs := buildTestTable(t, cidrs...)

	for i := range 20000 {
		var a [16]byte
		rng.Read(a[:])
		v6 := netip.AddrFrom16(a)
		ip := v6
		if i%2 == 0 {
			v4 := netip.AddrFrom4([4]byte{a[0], a[1], a[2], a[3]})
			ip = v4
		}
		s := ip.String()
		got, _ := table.MatchIP(s)
		want := oracleMatch(prefs, s)
		if got != want {
			t.Fatalf("MatchIP(%q) = %v, oracle = %v", s, got, want)
		}
	}
}

func TestCIDRTable_InsertOrderInvariant(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	base := []string{
		"253.128.0.0/10", "253.154.0.0/15", "252.241.207.0/24", "253.224.0.0/12",
		"10.0.0.0/8", "10.128.0.0/9", "10.0.0.0/16", "10.0.128.0/17",
	}
	for i := range 40 {
		var a [4]byte
		rng.Read(a[:])
		base = append(base, netip.PrefixFrom(netip.AddrFrom4(a), 8+rng.Intn(25)).Masked().String())
	}
	var probes []string
	prng := rand.New(rand.NewSource(7))
	for i := range 4000 {
		probes = append(probes, netip.AddrFrom4([4]byte{byte(prng.Intn(256)), byte(prng.Intn(256)), byte(prng.Intn(256)), byte(prng.Intn(256))}).String())
	}

	var want []bool
	for order := range 8 {
		sh := append([]string{}, base...)
		rng.Shuffle(len(sh), func(i, j int) { sh[i], sh[j] = sh[j], sh[i] })
		table, _ := buildTestTable(t, sh...)
		for pi, ip := range probes {
			got, _ := table.MatchIP(ip)
			if order == 0 {
				want = append(want, got)
				continue
			}
			if got != want[pi] {
				t.Fatalf("order %d: MatchIP(%q) = %v, want %v", order, ip, got, want[pi])
			}
		}
	}
}

func TestCIDRTable_OracleDenseSameLength(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	var cidrs []string

	for i := range 200 {
		p := netip.PrefixFrom(netip.AddrFrom4([4]byte{54, byte(rng.Intn(256)), byte(rng.Intn(256)), 0}), 24).Masked()
		cidrs = append(cidrs, p.String())
	}
	table, prefs := buildTestTable(t, cidrs...)
	for i := range 20000 {
		ip := netip.AddrFrom4([4]byte{54, byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256))}).String()
		got, _ := table.MatchIP(ip)
		if want := oracleMatch(prefs, ip); got != want {
			t.Fatalf("MatchIP(%q) = %v, oracle = %v", ip, got, want)
		}
	}
}

func TestCIDRTable_DuplicateInsert(t *testing.T) {
	table, _ := buildTestTable(t, "54.0.0.0/8", "54.0.0.0/8", "10.0.0.0/8", "54.0.0.0/8")
	if got, _ := table.MatchIP("54.9.9.9"); !got {
		t.Fatal("duplicate inserts must keep the prefix matchable")
	}
	n, _, ok := table.SnapshotSize()
	if !ok || n != 2 {
		t.Fatalf("SnapshotSize = %d, %v; want 2 unique prefixes", n, ok)
	}
}

func TestCIDRTable_FeedAttribution(t *testing.T) {
	table, _ := buildTestTable(t, "54.0.0.0/8", "2001:db8::/32")
	_, feed4 := table.MatchIP("54.1.1.1")
	if feed4 != CIDRFeedAWS {
		t.Fatalf("v4 feed = %d, want aws(%d)", feed4, CIDRFeedAWS)
	}
	_, feed6 := table.MatchIP("2001:db8::1")
	if feed6 != CIDRFeedTor {
		t.Fatalf("v6 feed = %d, want tor(%d)", feed6, CIDRFeedTor)
	}
}
