//go:build linux

package edge

import (
	"fmt"
	"net"

	"ad-event-processor/pkg/faultproof"

	"golang.org/x/sys/unix"
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
	if opts.Iface == "" {
		return IfaceResult{}, fmt.Errorf("iface required")
	}
	if opts.MalformedIters < 1 {
		opts.MalformedIters = 200
	}
	if opts.FloodPackets < 1 {
		opts.FloodPackets = 1000
	}
	if opts.Dst == nil {
		opts.Dst = net.IPv4(127, 0, 0, 1)
	}
	if opts.DPort == 0 {
		opts.DPort = TrackerPort
	}

	ifi, err := net.InterfaceByName(opts.Iface)
	if err != nil {
		return IfaceResult{}, err
	}

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		return IfaceResult{}, err
	}
	defer func() { _ = unix.Close(fd) }()

	addr := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ALL),
		Ifindex:  ifi.Index,
	}
	if err := unix.Bind(fd, addr); err != nil {
		return IfaceResult{}, err
	}

	dstMAC := ifi.HardwareAddr
	if len(dstMAC) != 6 {
		dstMAC = net.HardwareAddr{0, 0, 0, 0, 0, 0}
	}

	var res IfaceResult
	src := opts.Src
	if src == nil {
		src = net.IPv4(203, 0, 113, 77)
	}
	pkt := BuildSYNPacket(src, opts.Dst, opts.DPort)
	WithEthernetMAC(pkt, dstMAC, dstMAC)

	for i := 0; i < opts.MalformedIters; i++ {
		malformed := make([]byte, len(pkt))
		copy(malformed, pkt)
		malformed[i%len(malformed)] ^= 0xff
		if err := unix.Sendto(fd, malformed, 0, addr); err != nil {
			res.SendErrors++
			continue
		}
		res.SentMalformed++
	}

	for i := 0; i < opts.FloodPackets; i++ {
		if err := unix.Sendto(fd, pkt, 0, addr); err != nil {
			res.SendErrors++
			continue
		}
		res.SentFlood++
	}

	return res, nil
}

func EmitIfaceProof(iface string, res IfaceResult) {
	faultproof.Print("xdp_injector_iface", map[string]string{
		"iface":     iface,
		"malformed": fmt.Sprintf("%d", res.SentMalformed),
		"flood":     fmt.Sprintf("%d", res.SentFlood),
		"errors":    fmt.Sprintf("%d", res.SendErrors),
		"status":    "sent",
	})
}

func htons(v uint16) uint16 {
	return (v << 8) | (v >> 8)
}
