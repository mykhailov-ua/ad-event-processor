package edge

import (
	"net"
	"testing"

	"github.com/cilium/ebpf"
)

const benchOutputPad = 258

func benchRunOptions(pkt []byte) *ebpf.RunOptions {
	out := make([]byte, len(pkt)+benchOutputPad)
	return &ebpf.RunOptions{Data: pkt, DataOut: out, Repeat: 1}
}

func BenchmarkXDP_passSYN_noFingerprint(b *testing.B) {
	objs := loadBenchObjects(b)
	key := uint32(0)
	cfg := DefaultConfig(InitOptions{DisableFingerprint: true})
	if err := objs.Config.Update(&key, &cfg, ebpf.UpdateAny); err != nil {
		b.Fatal(err)
	}
	pkt := buildSYNPacketBench(net.IPv4(10, 1, 2, 3), trackerPort)
	opts := benchRunOptions(pkt)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := objs.XdpEdgeFilter.Run(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXDP_passSYN(b *testing.B) {
	objs := loadBenchObjects(b)
	pkt := buildSYNPacketBench(net.IPv4(10, 1, 2, 3), trackerPort)
	opts := benchRunOptions(pkt)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := objs.XdpEdgeFilter.Run(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXDP_dropBlocklist(b *testing.B) {
	objs := loadBenchObjects(b)
	if err := objs.BlocklistHostV4.Update(HostKey(10, 9, 8, 7).Addr, uint8(1), ebpf.UpdateAny); err != nil {
		b.Fatal(err)
	}
	pkt := buildSYNPacketBench(net.IPv4(10, 9, 8, 7), trackerPort)
	opts := benchRunOptions(pkt)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := objs.XdpEdgeFilter.Run(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXDP_passPPSACK(b *testing.B) {
	objs := loadBenchObjects(b)
	pkt := buildACKPacketBench(net.IPv4(10, 2, 3, 4), trackerPort)
	opts := benchRunOptions(pkt)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := objs.XdpEdgeFilter.Run(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXDP_dropAnomaly(b *testing.B) {
	objs := loadBenchObjects(b)
	pkt := buildSYNPacketBench(net.IPv4(10, 3, 4, 5), trackerPort)
	pkt[47] = 0x03
	opts := benchRunOptions(pkt)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := objs.XdpEdgeFilter.Run(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXDP_dropNonTCP(b *testing.B) {
	objs := loadBenchObjects(b)
	pkt := make([]byte, 42)
	pkt[12], pkt[13] = 0x08, 0x00
	ip := pkt[14:]
	ip[0] = 0x45
	ip[9] = 17
	copy(ip[12:16], []byte{10, 3, 4, 5})
	copy(ip[16:20], []byte{10, 0, 0, 1})
	udp := pkt[34:]
	udp[2] = byte(trackerPort >> 8)
	udp[3] = byte(trackerPort & 0xff)
	opts := benchRunOptions(pkt)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := objs.XdpEdgeFilter.Run(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func loadBenchObjects(b *testing.B) *EdgeObjects {
	b.Helper()
	if testing.Short() {
		b.Skip("skipping BPF bench in -short mode")
	}
	var objs EdgeObjects
	if err := LoadEdgeObjectsForTest(&objs, nil); err != nil {
		b.Skipf("BPF unavailable: %v", err)
	}
	_ = InitConfigWith(objs.Config, InitOptions{})
	_ = wireProgArrayEntries(&objs)
	b.Cleanup(func() { objs.Close() })
	return &objs
}

func buildSYNPacketBench(src net.IP, dport uint16) []byte {
	src4 := src.To4()
	pkt := make([]byte, 54)
	pkt[12], pkt[13] = 0x08, 0x00
	ip := pkt[14:]
	ip[0] = 0x45
	ip[9] = 6
	copy(ip[12:16], src4)
	copy(ip[16:20], []byte{10, 0, 0, 1})
	tcp := pkt[34:]
	tcp[12] = 0x50
	tcp[0] = 0x30
	tcp[1] = 0x39
	tcp[2] = byte(dport >> 8)
	tcp[3] = byte(dport)
	tcp[13] = 0x02
	return pkt
}

func buildACKPacketBench(src net.IP, dport uint16) []byte {
	pkt := buildSYNPacketBench(src, dport)
	pkt[47] = 0x10
	return pkt
}
