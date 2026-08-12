//go:build !linux

package faultinject

import (
	"fmt"
	"net"
)

// IfaceOptions configures raw-socket injection on a netdev (requires CAP_NET_RAW).
type IfaceOptions struct {
	Iface          string
	Src            net.IP
	MalformedIters int
	FloodPackets   int
	Dst            net.IP
	DPort          uint16
}

// IfaceResult summarizes iface-mode injection.
type IfaceResult struct {
	SentMalformed int
	SentFlood     int
	SendErrors    int
}

// RunIface is only supported on Linux.
func RunIface(opts IfaceOptions) (IfaceResult, error) {
	return IfaceResult{}, fmt.Errorf("iface injection requires linux")
}

// EmitIfaceProof is a no-op on non-Linux builds.
func EmitIfaceProof(iface string, res IfaceResult) {}
