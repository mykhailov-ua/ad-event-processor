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

// Tier-1 mobile network operator ASNs. Excludes reseller/proxy ASNs used by bot farms.
var builtinMobileCarrierASNs = map[uint32]struct{}{
	21928:  {}, // T-Mobile US
	310410: {}, // AT&T Mobility
	20057:  {}, // AT&T Services
	6167:   {}, // Verizon Business
	3215:   {}, // Orange
	12479:  {}, // Orange France
	3209:   {}, // Vodafone
	12956:  {}, // Telefonica
	3320:   {}, // Deutsche Telekom
	9808:   {}, // China Mobile
	58453:  {}, // China Mobile Hong Kong
	45400:  {}, // Telstra
	26615:  {}, // TIM Brasil
	2856:   {}, // BT Group
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
