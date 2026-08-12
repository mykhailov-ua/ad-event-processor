package faultinject

import (
	"encoding/binary"
	"net"
)

// TrackerPort is the default edge ingress TCP port exercised by the injector.
const TrackerPort = 8180

// BuildSYNPacket builds a minimal Ethernet+IPv4+TCP SYN frame for lab injection.
func BuildSYNPacket(src net.IP, dst net.IP, dport uint16) []byte {
	src4 := src.To4()
	dst4 := dst.To4()
	if src4 == nil || dst4 == nil {
		return nil
	}

	const (
		ethLen = 14
		ipLen  = 20
		tcpLen = 20
	)
	pkt := make([]byte, ethLen+ipLen+tcpLen)
	binary.BigEndian.PutUint16(pkt[12:14], 0x0800)

	ip := pkt[ethLen:]
	ip[0] = 0x45
	ip[9] = 6
	copy(ip[12:16], src4)
	copy(ip[16:20], dst4)

	tcp := pkt[ethLen+ipLen:]
	tcp[12] = 0x50
	binary.BigEndian.PutUint16(tcp[0:2], 12345)
	binary.BigEndian.PutUint16(tcp[2:4], dport)
	tcp[13] = 0x02

	return pkt
}

// WithEthernetMAC fills dst/src MAC addresses on a frame built by BuildSYNPacket.
func WithEthernetMAC(pkt []byte, dst, src net.HardwareAddr) {
	if len(pkt) < 14 {
		return
	}
	if len(dst) == 6 {
		copy(pkt[0:6], dst)
	}
	if len(src) == 6 {
		copy(pkt[6:12], src)
	}
}
