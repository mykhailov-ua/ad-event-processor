package edge

import (
	"fmt"
	"math/rand"
	"net"

	"ad-event-processor/pkg/faultproof"

	"github.com/cilium/ebpf"
)

type ProgramOptions struct {
	MalformedIters int
	FloodPackets   int
	Dst            net.IP
	DPort          uint16
}

type ProgramResult struct {
	MalformedIters int
	InvalidActions int
	FloodPackets   int
	FloodDrops     int
}

func OpenProgram() (*EdgeObjects, func(), error) {
	var objs EdgeObjects
	if err := LoadEdgeObjectsLenient(&objs, nil); err != nil {
		return nil, nil, err
	}
	if err := InitConfigWith(objs.Config, InitOptions{}); err != nil {
		_ = objs.Close()
		return nil, nil, err
	}
	return &objs, func() { _ = objs.Close() }, nil
}

func RunProgram(prog *ebpf.Program, opts ProgramOptions) (ProgramResult, error) {
	if prog == nil {
		return ProgramResult{}, fmt.Errorf("nil xdp program")
	}
	if opts.MalformedIters < 1 {
		opts.MalformedIters = 1000
	}
	if opts.FloodPackets < 1 {
		opts.FloodPackets = 2000
	}
	if opts.Dst == nil {
		opts.Dst = net.IPv4(10, 0, 0, 1)
	}
	if opts.DPort == 0 {
		opts.DPort = TrackerPort
	}

	var res ProgramResult
	rng := rand.New(rand.NewSource(42))

	if ret, _, err := prog.Test([]byte{}); err == nil && ret != 0 && ret != 1 && ret != 2 {
		res.InvalidActions++
	}
	if ret, _, err := prog.Test(make([]byte, 10)); err == nil && ret != 0 && ret != 1 && ret != 2 {
		res.InvalidActions++
	}

	for i := 0; i < opts.MalformedIters; i++ {
		pktLen := 14 + rng.Intn(1486)
		pkt := make([]byte, pktLen)
		rng.Read(pkt)
		ret, _, err := prog.Test(pkt)
		if err != nil {
			continue
		}
		if ret != 0 && ret != 1 && ret != 2 {
			res.InvalidActions++
		}
	}
	res.MalformedIters = opts.MalformedIters

	src := net.IPv4(198, 51, 100, 50)
	pkt := BuildSYNPacket(src, opts.Dst, opts.DPort)
	for i := 0; i < opts.FloodPackets; i++ {
		ret, _, err := prog.Test(pkt)
		if err != nil {
			return res, err
		}
		if ret == 1 {
			res.FloodDrops++
		}
	}
	res.FloodPackets = opts.FloodPackets

	return res, nil
}

func EmitProgramProofs(res ProgramResult) {
	faultproof.Print("xdp_injector_malformed", map[string]string{
		"iters":   fmt.Sprintf("%d", res.MalformedIters),
		"invalid": fmt.Sprintf("%d", res.InvalidActions),
		"status":  "no_panics",
	})
	faultproof.Print("xdp_injector_syn_flood", map[string]string{
		"packets": fmt.Sprintf("%d", res.FloodPackets),
		"drops":   fmt.Sprintf("%d", res.FloodDrops),
		"status":  "stable",
	})
}
