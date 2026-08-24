package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"ad-event-processor/internal/edge"
)

func main() {
	mode := flag.String("mode", "program", "Injection mode: program, iface, or all")
	iface := flag.String("iface", os.Getenv("INGRESS_INTERFACE"), "NIC for iface mode (requires CAP_NET_RAW)")
	iters := flag.Int("iters", 1000, "Malformed packet iterations")
	flood := flag.Int("flood", 2000, "High-rate SYN packet count")
	dport := flag.Int("dport", edge.TrackerPort, "Destination TCP port")
	dstIP := flag.String("dst", "10.0.0.1", "Destination IPv4 for crafted frames")
	flag.Parse()

	dst := net.ParseIP(*dstIP)
	if dst == nil {
		slog.Error("invalid dst IP", "dst", *dstIP)
		os.Exit(1)
	}

	switch *mode {
	case "program":
		if err := runProgram(*iters, *flood, dst, uint16(*dport)); err != nil {
			slog.Error("program injection failed", "error", err)
			os.Exit(1)
		}
	case "iface":
		if *iface == "" {
			slog.Error("iface required for iface mode (-iface or INGRESS_INTERFACE)")
			os.Exit(1)
		}
		if err := runIface(*iface, *iters, *flood, dst, uint16(*dport)); err != nil {
			slog.Error("iface injection failed", "error", err)
			os.Exit(1)
		}
	case "all":
		if err := runProgram(*iters, *flood, dst, uint16(*dport)); err != nil {
			slog.Error("program injection failed", "error", err)
			os.Exit(1)
		}
		if *iface == "" {
			slog.Warn("iface mode skipped (no -iface)")
			return
		}
		if err := runIface(*iface, *iters/5, *flood/2, dst, uint16(*dport)); err != nil {
			slog.Error("iface injection failed", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", *mode)
		os.Exit(2)
	}
}

func runProgram(iters, flood int, dst net.IP, dport uint16) error {
	objs, cleanup, err := edge.OpenProgram()
	if err != nil {
		return err
	}
	defer cleanup()

	res, err := edge.RunProgram(objs.XdpEdgeFilter, edge.ProgramOptions{
		MalformedIters: iters,
		FloodPackets:   flood,
		Dst:            dst,
		DPort:          dport,
	})
	if err != nil {
		return err
	}
	edge.EmitProgramProofs(res)
	slog.Info("program injection complete",
		"malformed_iters", res.MalformedIters,
		"flood_packets", res.FloodPackets,
		"flood_drops", res.FloodDrops,
	)
	return nil
}

func runIface(iface string, iters, flood int, dst net.IP, dport uint16) error {
	res, err := edge.RunIface(edge.IfaceOptions{
		Iface:          iface,
		MalformedIters: iters,
		FloodPackets:   flood,
		Dst:            dst,
		DPort:          dport,
	})
	if err != nil {
		return err
	}
	edge.EmitIfaceProof(iface, res)
	slog.Info("iface injection complete",
		"iface", iface,
		"malformed", res.SentMalformed,
		"flood", res.SentFlood,
		"errors", res.SendErrors,
	)
	return nil
}
