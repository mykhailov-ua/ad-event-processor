package bpf

import (
	"fmt"

	"github.com/cilium/ebpf"
)

const (
	StatPass = iota
	StatPassAllowlist
	StatDropBlocklist
	StatDropSyn
	StatDropGlobalSyn
	StatDropPPS
	StatDropAnomaly
	StatDropInvalid
	StatDropNonTCP
	StatDropRST
	StatDropSynSubnet
	StatSynCookie
	StatFingerprint
	StatMax
)

func StatReason(idx uint32) string {
	switch idx {
	case StatPass:
		return "pass"
	case StatPassAllowlist:
		return "pass_allowlist"
	case StatDropBlocklist:
		return "blocklist"
	case StatDropSyn:
		return "syn"
	case StatDropGlobalSyn:
		return "global_syn"
	case StatDropPPS:
		return "pps"
	case StatDropAnomaly:
		return "anomaly"
	case StatDropInvalid:
		return "invalid"
	case StatDropNonTCP:
		return "non_tcp"
	case StatDropRST:
		return "rst"
	case StatDropSynSubnet:
		return "syn_subnet"
	case StatSynCookie:
		return "syn_cookie"
	case StatFingerprint:
		return "fingerprint"
	default:
		return fmt.Sprintf("unknown_%d", idx)
	}
}

func AggregateStats(m *ebpf.Map) ([]uint64, error) {
	if m == nil {
		return nil, fmt.Errorf("stats map is nil")
	}
	out := make([]uint64, StatMax)
	for idx := uint32(0); idx < StatMax; idx++ {
		var perCPU []uint64
		if err := m.Lookup(&idx, &perCPU); err != nil {
			return nil, fmt.Errorf("lookup stat %d: %w", idx, err)
		}
		var sum uint64
		for _, v := range perCPU {
			sum += v
		}
		out[idx] = sum
	}
	return out, nil
}
