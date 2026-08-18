//go:build !linux

package edge

import (
	"fmt"
	"net"
)

type IfaceOptions struct {
	Iface          string
	Src            net.IP
	MalformedIters int
	FloodPackets   int
	Dst            net.IP
	DPort          uint16
}

type IfaceResult struct {
	SentMalformed int
	SentFlood     int
	SendErrors    int
}

func RunIface(opts IfaceOptions) (IfaceResult, error) {
	return IfaceResult{}, fmt.Errorf("iface injection requires linux")
}

func EmitIfaceProof(iface string, res IfaceResult) {}
