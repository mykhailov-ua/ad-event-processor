package ingestion

import (
	"net/netip"
	"sync/atomic"
)

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
	nodes []cidrNode
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

func proxyVPNConnTypeBlocks(connType uint8) bool {
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
