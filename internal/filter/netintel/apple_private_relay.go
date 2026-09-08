package netintel

import (
	"strings"
	"sync/atomic"
)

// ApplePrivateRelayTable holds relay egress ASNs (Cloudflare/Fastly iCloud Private Relay).
// Cold reload: netintel:apple_relay:v1 in Redis (future); builtin seed for dev.
type ApplePrivateRelayTable struct {
	active atomic.Pointer[appleRelaySnapshot]
}

type appleRelaySnapshot struct {
	asn map[uint32]struct{}
}

func NewApplePrivateRelayTable(extra map[uint32]struct{}) *ApplePrivateRelayTable {
	t := &ApplePrivateRelayTable{}
	t.Publish(mergeBuiltinAppleRelayASNs(extra))
	return t
}

func (t *ApplePrivateRelayTable) Publish(asn map[uint32]struct{}) {
	if t == nil {
		return
	}
	dup := make(map[uint32]struct{}, len(asn))
	for k := range asn {
		dup[k] = struct{}{}
	}
	t.active.Store(&appleRelaySnapshot{asn: dup})
}

func (t *ApplePrivateRelayTable) IsRelayASN(asn uint32) bool {
	if t == nil || asn == 0 {
		return false
	}
	snap := t.active.Load()
	if snap == nil {
		return false
	}
	_, ok := snap.asn[asn]
	return ok
}

// ExemptFromDCASN returns true when relay ASN pairs with an Apple-family UA (skip datacenter_ip only).
func (t *ApplePrivateRelayTable) ExemptFromDCASN(asn uint32, ua string) bool {
	if !t.IsRelayASN(asn) {
		return false
	}
	if uaLooksApplePrivateRelay(ua) {
		return true
	}
	return false
}

func uaLooksApplePrivateRelay(ua string) bool {
	if ua == "" {
		return false
	}
	n := len(ua)
	if n > 512 {
		n = 512
	}
	s := ua[:n]
	if strings.Contains(s, "iPhone") || strings.Contains(s, "iPad") || strings.Contains(s, "Macintosh") {
		return true
	}
	if strings.Contains(s, "CPU iPhone OS") || strings.Contains(s, "CPU OS") {
		return true
	}
	return false
}

var builtinAppleRelayASNs = map[uint32]struct{}{
	13335: {}, // Cloudflare
	54113: {}, // Fastly
}

func mergeBuiltinAppleRelayASNs(extra map[uint32]struct{}) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(builtinAppleRelayASNs)+len(extra))
	for asn := range builtinAppleRelayASNs {
		out[asn] = struct{}{}
	}
	for asn := range extra {
		if asn != 0 {
			out[asn] = struct{}{}
		}
	}
	return out
}
