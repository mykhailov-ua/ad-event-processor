package netintel

import (
	"bufio"
	"context"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

type proxyVPNBuilder struct {
	nodes []CIDRNode
	prefs []proxyVPNPrefix
}

type ProxyVPNBuilder = proxyVPNBuilder

func (b *ProxyVPNBuilder) AddPrefix(p netip.Prefix, connType uint8, asn uint32, root4, root6 *int32) {
	b.addPrefix(p, connType, asn, root4, root6)
}

func (b *ProxyVPNBuilder) Snapshot(root4, root6 int32, gen uint64) *ProxyVPNSnapshot {
	return b.snapshot(root4, root6, gen)
}

type ProxyVPNSnapshot = proxyVPNSnapshot

func (b *proxyVPNBuilder) addPrefix(p netip.Prefix, connType uint8, asn uint32, root4, root6 *int32) {
	if p.Addr().Is4() {
		a4 := p.Addr().As4()
		var key [16]byte
		copy(key[:4], a4[:])
		b.insert(root4, key, uint8(p.Bits()), connType, asn)
		return
	}
	b.insert(root6, p.Addr().As16(), uint8(p.Bits()), connType, asn)
}

func (b *proxyVPNBuilder) addLeaf(pref int32) int32 {
	b.nodes = append(b.nodes, CIDRNode{
		child:   [2]int32{cidrNoIndex, cidrNoIndex},
		pref:    pref,
		critbit: -1,
	})
	return int32(len(b.nodes) - 1)
}

func (b *proxyVPNBuilder) repPrefix(cur int32) proxyVPNPrefix {
	for b.nodes[cur].critbit >= 0 {
		next := b.nodes[cur].child[0]
		if next < 0 {
			next = b.nodes[cur].child[1]
		}
		cur = next
	}
	return b.prefs[b.nodes[cur].pref]
}

func (b *proxyVPNBuilder) insert(root *int32, addr [16]byte, nbits uint8, connType uint8, asn uint32) {
	prefIdx := int32(len(b.prefs))
	b.prefs = append(b.prefs, proxyVPNPrefix{addr: addr, bits: nbits, connType: connType, asn: asn})

	if *root < 0 {
		*root = b.addLeaf(prefIdx)
		return
	}

	parent := int32(cidrNoIndex)
	var pdir uint32
	cur := *root
	missing := false
	for b.nodes[cur].critbit >= 0 && b.nodes[cur].critbit < int32(nbits) {
		parent = cur
		pdir = cidrBitAt(&addr, b.nodes[cur].critbit)
		next := b.nodes[cur].child[pdir]
		if next < 0 {
			missing = true
			break
		}
		cur = next
	}

	rep := b.repPrefix(cur)
	d := cidrFirstDiff(&addr, nbits, &rep.addr, rep.bits)
	if missing {
		if d < 0 || d >= b.nodes[cur].critbit {
			b.nodes[cur].child[pdir] = b.addLeaf(prefIdx)
			return
		}
	}
	if d < 0 {
		switch {
		case nbits == rep.bits:
			b.prefs = b.prefs[:prefIdx]
			return
		case nbits < rep.bits:
			if b.nodes[cur].critbit == int32(nbits) {
				if b.nodes[cur].pref < 0 {
					b.nodes[cur].pref = prefIdx
				} else {
					b.prefs = b.prefs[:prefIdx]
				}
				return
			}
			d = int32(nbits)
		default:
			repLeaf := b.nodes[cur]
			split := b.addNode(int32(rep.bits), repLeaf.pref)
			b.nodes[split].child[cidrBitAt(&addr, int32(rep.bits))] = b.addLeaf(prefIdx)
			b.linkChild(root, parent, pdir, split)
			return
		}
	}

	if d >= int32(nbits) {
		split := b.addNode(d, prefIdx)
		b.nodes[split].child[cidrBitAt(&rep.addr, d)] = cur
		b.linkChild(root, parent, pdir, split)
		return
	}

	walk := *root
	wparent := int32(cidrNoIndex)
	var wpdir uint32
	for b.nodes[walk].critbit >= 0 && b.nodes[walk].critbit < d {
		wparent = walk
		wpdir = cidrBitAt(&addr, b.nodes[walk].critbit)
		walk = b.nodes[walk].child[wpdir]
	}
	split := b.addNode(d, cidrNoIndex)
	kdir := cidrBitAt(&addr, d)
	b.nodes[split].child[kdir] = b.addLeaf(prefIdx)
	b.nodes[split].child[1-kdir] = walk
	if wparent < 0 {
		*root = split
		return
	}
	b.nodes[wparent].child[wpdir] = split
}

func (b *proxyVPNBuilder) snapshot(root4, root6 int32, gen uint64) *proxyVPNSnapshot {
	return &proxyVPNSnapshot{
		gen:   gen,
		root4: root4,
		root6: root6,
		nodes: b.nodes,
		prefs: b.prefs,
	}
}

func (b *proxyVPNBuilder) addNode(critbit, pref int32) int32 {
	b.nodes = append(b.nodes, CIDRNode{
		child:   [2]int32{cidrNoIndex, cidrNoIndex},
		pref:    pref,
		critbit: critbit,
	})
	return int32(len(b.nodes) - 1)
}

func (b *proxyVPNBuilder) linkChild(root *int32, parent int32, pdir uint32, node int32) {
	if parent < 0 {
		*root = node
		return
	}
	b.nodes[parent].child[pdir] = node
}

type proxyVPNFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *ProxyVPNTable
	gen     atomic.Uint64
}

func NewProxyVPNFeedLoader(cfg *config.Config, table *ProxyVPNTable) *proxyVPNFeedLoader {
	if cfg == nil || !cfg.ProxyVPNBlockEnabled || table == nil {
		return nil
	}
	l := &proxyVPNFeedLoader{
		dir:     cfg.ProxyVPNFeedDir,
		refresh: cfg.ProxyVPNFeedRefresh,
		table:   table,
	}
	if l.refresh <= 0 {
		l.refresh = 24 * time.Hour
	}
	if l.dir == "" {
		l.dir = "/var/lib/ad-event-processor/proxy-vpn"
	}
	return l
}

func (l *proxyVPNFeedLoader) Start(ctx context.Context) {
	if l == nil {
		return
	}
	l.reloadOnce()
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.reloadOnce()
		}
	}
}

func (l *proxyVPNFeedLoader) reloadOnce() {
	path := filepath.Join(l.dir, "proxy_vpn.txt")
	lines := l.scanFeedFile(path)
	extPath := filepath.Join(l.dir, "external_residential.txt")
	lines += l.scanFeedFile(extPath)
	if lines == 0 {
		metrics.ProxyVPNLPMUninitialized.Set(1)
		return
	}
	gen := l.gen.Add(1)
	var b proxyVPNBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	f, err := os.Open(path)
	if err == nil {
		defer func() { _ = f.Close() }()
		l.scanFeedIntoBuilder(bufio.NewScanner(f), &b, &root4, &root6)
	}
	if ef, err := os.Open(extPath); err == nil {
		defer func() { _ = ef.Close() }()
		l.scanFeedIntoBuilder(bufio.NewScanner(ef), &b, &root4, &root6)
	}
	snap := b.snapshot(root4, root6, gen)
	l.table.Publish(snap)
	metrics.ProxyVPNFeedRefreshTotal.Inc()
	metrics.ProxyVPNLPMPrefixes.Set(float64(len(snap.prefs)))
	metrics.ProxyVPNLPMUninitialized.Set(0)
	slog.Info("proxy vpn feed published", "prefixes", len(snap.prefs), "gen", gen, "lines", lines)
}

func (l *proxyVPNFeedLoader) scanFeedFile(path string) int {
	f, err := os.Open(path)
	if err != nil {
		if path == filepath.Join(l.dir, "proxy_vpn.txt") {
			metrics.ProxyVPNFeedRefreshErrorsTotal.Inc()
			slog.Warn("proxy vpn feed open failed", "path", path, "error", err)
		}
		return 0
	}
	defer func() { _ = f.Close() }()
	return l.scanFeedIntoBuilder(bufio.NewScanner(f), nil, nil, nil)
}

func (l *proxyVPNFeedLoader) scanFeedIntoBuilder(sc *bufio.Scanner, b *proxyVPNBuilder, root4, root6 *int32) int {
	lines := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, connType, asn, ok := ParseProxyVPNFeedLine(line)
		if !ok {
			continue
		}
		if b != nil {
			b.addPrefix(prefix, connType, asn, root4, root6)
		}
		lines++
	}
	if err := sc.Err(); err != nil {
		metrics.ProxyVPNFeedRefreshErrorsTotal.Inc()
		slog.Warn("proxy vpn feed scan failed", "error", err)
		return 0
	}
	return lines
}

func ParseProxyVPNFeedLine(line string) (netip.Prefix, uint8, uint32, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return netip.Prefix{}, 0, 0, false
	}
	prefix, err := netip.ParsePrefix(fields[0])
	if err != nil {
		return netip.Prefix{}, 0, 0, false
	}
	if !prefix.IsValid() {
		return netip.Prefix{}, 0, 0, false
	}
	prefix = prefix.Masked()
	var asn uint32
	if fields[1] != "-" && fields[1] != "0" {
		n, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return netip.Prefix{}, 0, 0, false
		}
		asn = uint32(n)
	}
	flags := ""
	if len(fields) > 2 {
		flags = strings.Join(fields[2:], " ")
	}
	connType := parseProxyVPNConnFlags(flags)
	if connType == 0 {
		connType = ProxyVPNConnVPN | ProxyVPNConnHosting
	}
	return prefix, connType, asn, true
}

const (
	ProxyVPNConnISP uint8 = 1 << iota
	ProxyVPNConnHosting
	ProxyVPNConnVPN
	ProxyVPNConnMobile
)

type proxyVPNPrefix struct {
	addr     [16]byte
	bits     uint8
	connType uint8
	asn      uint32
}

type proxyVPNSnapshot struct {
	gen   uint64
	root4 int32
	root6 int32
	nodes []CIDRNode
	prefs []proxyVPNPrefix
}

type ProxyVPNTable struct {
	active atomic.Pointer[proxyVPNSnapshot]
}

func NewProxyVPNTable() *ProxyVPNTable {
	return &ProxyVPNTable{}
}

func (t *ProxyVPNTable) Publish(s *proxyVPNSnapshot) {
	t.active.Store(s)
}

func (t *ProxyVPNTable) Ready() bool {
	return t.active.Load() != nil
}

func (t *ProxyVPNTable) SnapshotSize() (prefixes int, gen uint64, ok bool) {
	snap := t.active.Load()
	if snap == nil {
		return 0, 0, false
	}
	return len(snap.prefs), snap.gen, true
}

func (t *ProxyVPNTable) Lookup4(ip [4]byte) (match bool, connType uint8, asn uint32) {
	snap := t.active.Load()
	if snap == nil || snap.root4 < 0 {
		return false, 0, 0
	}
	var key [16]byte
	copy(key[:4], ip[:])
	return snap.lookup(snap.root4, &key)
}

func (t *ProxyVPNTable) Lookup6(ip [16]byte) (match bool, connType uint8, asn uint32) {
	snap := t.active.Load()
	if snap == nil || snap.root6 < 0 {
		return false, 0, 0
	}
	return snap.lookup(snap.root6, &ip)
}

func (t *ProxyVPNTable) MatchIP(ipStr string) (match bool, connType uint8, asn uint32) {
	snap := t.active.Load()
	if snap == nil {
		return false, 0, 0
	}
	a, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false, 0, 0
	}
	if a.Is4() || a.Is4In6() {
		a4 := a.As4()
		if snap.root4 < 0 {
			return false, 0, 0
		}
		var key [16]byte
		copy(key[:4], a4[:])
		return snap.lookup(snap.root4, &key)
	}
	if snap.root6 < 0 {
		return false, 0, 0
	}
	key := a.As16()
	return snap.lookup(snap.root6, &key)
}

func (vp *proxyVPNSnapshot) lookup(root int32, key *[16]byte) (bool, uint8, uint32) {
	nodes := vp.nodes
	prefs := vp.prefs
	_ = nodes[len(nodes)-1]
	cur := root
	var bestConn uint8
	var bestASN uint32
	var matched bool
	for cur >= 0 {
		n := &nodes[cur]
		if n.pref >= 0 {
			p := &prefs[n.pref]
			if cidrPrefixMatch(key, &p.addr, p.bits) {
				matched = true
				bestConn = p.connType
				bestASN = p.asn
			}
		}
		if n.critbit < 0 {
			break
		}
		cur = n.child[cidrBitAt(key, n.critbit)]
	}
	return matched, bestConn, bestASN
}

func ProxyVPNConnTypeBlocks(connType uint8) bool {
	return connType&(ProxyVPNConnVPN|ProxyVPNConnHosting) != 0
}

func parseProxyVPNConnFlags(s string) uint8 {
	var mask uint8
	for part := range splitConnFlagParts(s) {
		switch part {
		case "isp":
			mask |= ProxyVPNConnISP
		case "hosting", "host", "dc", "datacenter":
			mask |= ProxyVPNConnHosting
		case "vpn", "proxy":
			mask |= ProxyVPNConnVPN
		case "mobile", "mob":
			mask |= ProxyVPNConnMobile
		}
	}
	return mask
}

func splitConnFlagParts(s string) func(func(string) bool) {
	return func(yield func(string) bool) {
		start := 0
		for i := 0; i <= len(s); i++ {
			if i == len(s) || s[i] == ',' {
				part := trimASCII(s[start:i])
				if part != "" && !yield(part) {
					return
				}
				start = i + 1
			}
		}
	}
}

func trimASCII(s string) string {
	for len(s) > 0 && s[0] <= ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] <= ' ' {
		s = s[:len(s)-1]
	}
	return s
}
