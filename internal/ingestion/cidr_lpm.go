package ingestion

import (
	"math/bits"
	"net/netip"
	"sync/atomic"
)

const (
	CIDRFeedAWS uint8 = iota
	CIDRFeedGCP
	CIDRFeedAzure
	CIDRFeedTor
	CIDRFeedOther
	CIDRFeedCount
)

var cidrFeedNames = [CIDRFeedCount]string{"aws", "gcp", "azure", "tor", "other"}

const cidrNoIndex = -1

type cidrNode struct {
	child   [2]int32
	pref    int32
	critbit int32
}

type cidrPrefix struct {
	addr [16]byte
	bits uint8
	feed uint8
}

type cidrSnapshot struct {
	gen   uint64
	root4 int32
	root6 int32
	nodes []cidrNode
	prefs []cidrPrefix
}

type CIDRTable struct {
	active atomic.Pointer[cidrSnapshot]
}

func NewCIDRTable() *CIDRTable {
	return &CIDRTable{}
}

func (t *CIDRTable) Publish(s *cidrSnapshot) {
	t.active.Store(s)
}

func (t *CIDRTable) Ready() bool {
	return t.active.Load() != nil
}

func (t *CIDRTable) SnapshotSize() (prefixes int, gen uint64, ok bool) {
	snap := t.active.Load()
	if snap == nil {
		return 0, 0, false
	}
	return len(snap.prefs), snap.gen, true
}

func (t *CIDRTable) Match4(ip [4]byte) (bool, uint8) {
	snap := t.active.Load()
	if snap == nil || snap.root4 < 0 {
		return false, 0
	}
	var key [16]byte
	copy(key[:4], ip[:])
	return snap.lookup(snap.root4, &key)
}

func (t *CIDRTable) Match6(ip [16]byte) (bool, uint8) {
	snap := t.active.Load()
	if snap == nil || snap.root6 < 0 {
		return false, 0
	}
	return snap.lookup(snap.root6, &ip)
}

func (t *CIDRTable) MatchIP(ipStr string) (bool, uint8) {
	snap := t.active.Load()
	if snap == nil {
		return false, 0
	}
	a, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false, 0
	}
	if a.Is4() || a.Is4In6() {
		a4 := a.As4()
		if snap.root4 < 0 {
			return false, 0
		}
		var key [16]byte
		copy(key[:4], a4[:])
		return snap.lookup(snap.root4, &key)
	}
	if snap.root6 < 0 {
		return false, 0
	}
	key := a.As16()
	return snap.lookup(snap.root6, &key)
}

func (s *cidrSnapshot) lookup(root int32, key *[16]byte) (bool, uint8) {
	nodes := s.nodes
	prefs := s.prefs
	_ = nodes[len(nodes)-1]
	cur := root
	for cur >= 0 {
		n := &nodes[cur]
		if n.pref >= 0 {
			p := &prefs[n.pref]
			if cidrPrefixMatch(key, &p.addr, p.bits) {
				return true, p.feed
			}
		}
		if n.critbit < 0 {
			return false, 0
		}
		cur = n.child[cidrBitAt(key, n.critbit)]
	}
	return false, 0
}

func cidrBitAt(key *[16]byte, i int32) uint32 {
	return uint32(key[i>>3]>>(7-(i&7))) & 1
}

func cidrPrefixMatch(key, addr *[16]byte, nbits uint8) bool {
	n := int(nbits)
	full := n >> 3
	for i := 0; i < full; i++ {
		if key[i] != addr[i] {
			return false
		}
	}
	rem := n & 7
	if rem == 0 {
		return true
	}
	mask := byte(0xff << (8 - rem))
	return key[full]&mask == addr[full]&mask
}

func cidrFirstDiff(a *[16]byte, abits uint8, b *[16]byte, bbits uint8) int32 {
	n := abits
	if bbits < n {
		n = bbits
	}
	full := int(n) >> 3
	for i := 0; i < full; i++ {
		if a[i] != b[i] {
			return int32(i*8 + bits.LeadingZeros8(a[i]^b[i]))
		}
	}
	rem := n & 7
	if rem > 0 {
		mask := byte(0xff << (8 - rem))
		if x := (a[full] ^ b[full]) & mask; x != 0 {
			return int32(full*8 + bits.LeadingZeros8(x))
		}
	}
	return -1
}

type cidrBuilder struct {
	nodes []cidrNode
	prefs []cidrPrefix
}

func (b *cidrBuilder) addPrefix(p netip.Prefix, feed uint8, root4, root6 *int32) {
	if p.Addr().Is4() {
		a4 := p.Addr().As4()
		var key [16]byte
		copy(key[:4], a4[:])
		b.insert(root4, key, uint8(p.Bits()), feed)
		return
	}
	b.insert(root6, p.Addr().As16(), uint8(p.Bits()), feed)
}

func (b *cidrBuilder) addLeaf(pref int32) int32 {
	b.nodes = append(b.nodes, cidrNode{
		child:   [2]int32{cidrNoIndex, cidrNoIndex},
		pref:    pref,
		critbit: -1,
	})
	return int32(len(b.nodes) - 1)
}

func (b *cidrBuilder) repPrefix(cur int32) cidrPrefix {
	for b.nodes[cur].critbit >= 0 {
		next := b.nodes[cur].child[0]
		if next < 0 {
			next = b.nodes[cur].child[1]
		}
		cur = next
	}
	return b.prefs[b.nodes[cur].pref]
}

func (b *cidrBuilder) insert(root *int32, addr [16]byte, nbits uint8, feed uint8) {
	prefIdx := int32(len(b.prefs))
	b.prefs = append(b.prefs, cidrPrefix{addr: addr, bits: nbits, feed: feed})

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

func (b *cidrBuilder) snapshot(root4, root6 int32, gen uint64) *cidrSnapshot {
	return &cidrSnapshot{
		gen:   gen,
		root4: root4,
		root6: root6,
		nodes: b.nodes,
		prefs: b.prefs,
	}
}

func (b *cidrBuilder) addNode(critbit, pref int32) int32 {
	b.nodes = append(b.nodes, cidrNode{
		child:   [2]int32{cidrNoIndex, cidrNoIndex},
		pref:    pref,
		critbit: critbit,
	})
	return int32(len(b.nodes) - 1)
}

func (b *cidrBuilder) linkChild(root *int32, parent int32, pdir uint32, node int32) {
	if parent < 0 {
		*root = node
		return
	}
	b.nodes[parent].child[pdir] = node
}
