package ingestion

import (
	"ad-event-processor/pkg/moderatorintel"
)

type ModeratorIPTable struct {
	table *CIDRTable
}

func NewModeratorIPTable() *ModeratorIPTable {
	return &ModeratorIPTable{table: NewCIDRTable()}
}

func (t *ModeratorIPTable) Ready() bool {
	return t != nil && t.table != nil && t.table.Ready()
}

func (t *ModeratorIPTable) MatchIP(ip string) (bool, uint8) {
	if t == nil || t.table == nil {
		return false, 0
	}
	return t.table.MatchIP(ip)
}

func (t *ModeratorIPTable) publishEntries(entries []moderatorintel.Entry, gen uint64) {
	if t == nil || t.table == nil || len(entries) == 0 {
		return
	}
	var b cidrBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for _, row := range entries {
		b.addPrefix(row.Prefix, row.Network, &root4, &root6)
	}
	if len(b.prefs) == 0 {
		return
	}
	t.table.Publish(b.snapshot(root4, root6, gen))
}
