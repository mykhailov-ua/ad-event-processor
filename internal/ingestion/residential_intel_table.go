package ingestion

import "net/netip"

type ResidentialIntelTable struct {
	table *CIDRTable
}

func NewResidentialIntelTable() *ResidentialIntelTable {
	return &ResidentialIntelTable{table: NewCIDRTable()}
}

func (t *ResidentialIntelTable) Ready() bool {
	return t != nil && t.table != nil && t.table.Ready()
}

func (t *ResidentialIntelTable) MatchIP(ip string) bool {
	if t == nil || t.table == nil {
		return false
	}
	match, _ := t.table.MatchIP(ip)
	return match
}

func (t *ResidentialIntelTable) publishPrefixes(prefixes []netip.Prefix, gen uint64) {
	if t == nil || t.table == nil || len(prefixes) == 0 {
		return
	}
	var b cidrBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for _, p := range prefixes {
		if !p.IsValid() {
			continue
		}
		b.addPrefix(p.Masked(), CIDRFeedOther, &root4, &root6)
	}
	if len(b.prefs) == 0 {
		return
	}
	t.table.Publish(b.snapshot(root4, root6, gen))
}
