package ingestion

import (
	"net/netip"
)

type proxyVPNBuilder struct {
	nodes []cidrNode
	prefs []proxyVPNPrefix
}

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
	b.nodes = append(b.nodes, cidrNode{
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
	b.nodes = append(b.nodes, cidrNode{
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
