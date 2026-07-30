package bpf

import (
	"encoding/binary"
	"fmt"
	"net"
)

func HostIPv4(addr uint32) string {
	return net.IPv4(
		byte(addr),
		byte(addr>>8),
		byte(addr>>16),
		byte(addr>>24),
	).String()
}

func WireIPv4(ip string) (uint32, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return 0, fmt.Errorf("invalid ip %q", ip)
	}
	v4 := parsed.To4()
	if v4 == nil {
		return 0, fmt.Errorf("not ipv4 %q", ip)
	}
	return binary.LittleEndian.Uint32(v4), nil
}

func ViolationReasonLabel(reason uint8) string {
	switch reason {
	case ViolationSYN:
		return "syn"
	case ViolationGlobalSYN:
		return "global_syn"
	case ViolationPPS:
		return "pps"
	case ViolationSYNSubnet:
		return "syn_subnet"
	default:
		return fmt.Sprintf("unknown_%d", reason)
	}
}
