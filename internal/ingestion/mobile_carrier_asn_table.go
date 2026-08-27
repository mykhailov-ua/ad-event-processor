package ingestion

import (
	"strings"
	"sync/atomic"
)

type mobileCarrierASNSnapshot struct {
	asn map[uint32]struct{}
}

type MobileCarrierASNTable struct {
	active atomic.Pointer[mobileCarrierASNSnapshot]
}

func NewMobileCarrierASNTable(extra map[uint32]struct{}) *MobileCarrierASNTable {
	t := &MobileCarrierASNTable{}
	t.Publish(mergeBuiltinMobileCarrierASNs(extra))
	return t
}

func (t *MobileCarrierASNTable) Publish(asn map[uint32]struct{}) {
	if t == nil {
		return
	}
	if len(asn) == 0 {
		t.active.Store(&mobileCarrierASNSnapshot{asn: map[uint32]struct{}{}})
		return
	}
	dup := make(map[uint32]struct{}, len(asn))
	for k := range asn {
		dup[k] = struct{}{}
	}
	t.active.Store(&mobileCarrierASNSnapshot{asn: dup})
}

func (t *MobileCarrierASNTable) IsMobileCarrier(asn uint32) bool {
	if t == nil || asn == 0 {
		return false
	}
	snap := t.active.Load()
	if snap == nil || len(snap.asn) == 0 {
		return false
	}
	_, ok := snap.asn[asn]
	return ok
}

func mergeBuiltinMobileCarrierASNs(extra map[uint32]struct{}) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(builtinMobileCarrierASNs)+len(extra))
	for asn := range builtinMobileCarrierASNs {
		out[asn] = struct{}{}
	}
	for asn := range extra {
		if asn != 0 {
			out[asn] = struct{}{}
		}
	}
	return out
}

var builtinMobileCarrierASNs = map[uint32]struct{}{
	21928:  {},
	310410: {},
	20057:  {},
	6167:   {},
	3215:   {},
	12479:  {},
	3209:   {},
	12956:  {},
	3320:   {},
	9808:   {},
	58453:  {},
	45400:  {},
	26615:  {},
	2856:   {},
}

func ParseMobileCarrierASNs(raw string) map[uint32]struct{} {
	if raw == "" {
		return nil
	}
	out := make(map[uint32]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if asn, ok := parseASNLine(part); ok {
			out[asn] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
