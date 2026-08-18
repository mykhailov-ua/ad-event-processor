package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
	"github.com/bidshard/ad-event-processor/pkg/netaddr"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/redis/go-redis/v9"
)

func main() {
	iface := flag.String("iface", os.Getenv("INGRESS_INTERFACE"), "network interface for XDP attach")
	pinDir := flag.String("pin-dir", edge.BPFPinDir(), "directory for pinned BPF maps")
	mode := flag.String("mode", edge.EnvOr("XDP_MODE", "generic"), "XDP attach mode: generic|native|offload")
	flag.Parse()

	if *iface == "" {
		slog.Error("INGRESS_INTERFACE or -iface is required")
		os.Exit(1)
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		slog.Error("rlimit remove memlock", "error", err)
		os.Exit(1)
	}

	if _, err := net.InterfaceByName(*iface); err != nil {
		slog.Error("interface lookup failed", "iface", *iface, "error", err)
		os.Exit(1)
	}

	objs := edge.EdgeObjects{}
	if err := edge.LoadEdgeObjectsLenient(&objs, nil); err != nil {
		slog.Error("load bpf objects", "error", err)
		os.Exit(1)
	}
	defer func() { _ = objs.Close() }()

	if err := edge.InitConfigFromEnv(objs.Config); err != nil {
		slog.Error("init bpf config", "error", err)
		os.Exit(1)
	}
	if err := wireProgArray(&objs); err != nil {
		slog.Error("wire prog array", "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*pinDir, 0o755); err != nil {
		slog.Error("mkdir pin dir", "path", *pinDir, "error", err)
		os.Exit(1)
	}
	if err := pinMaps(&objs, *pinDir); err != nil {
		slog.Error("pin maps", "error", err)
		os.Exit(1)
	}

	var xdpLink link.Link
	if ebpfEdgeAttachAllowed() {
		var attachedMode string
		var err error
		xdpLink, attachedMode, err = attachXDPWithFallback(*iface, objs.XdpEdgeFilter, *mode)
		if err != nil {
			slog.Error("attach xdp", "iface", *iface, "mode", *mode, "error", err)
			os.Exit(1)
		}
		defer func() { _ = xdpLink.Close() }()
		slog.Info("edge xdp attached", "iface", *iface, "mode", attachedMode, "requested_mode", *mode, "pin_dir", *pinDir, "syn_cookie", edge.SynCookieEnabled())
	} else {
		slog.Warn("ebpf_xdp_edge module not licensed; skipping XDP attach (maps pinned)", "iface", *iface, "pin_dir", *pinDir)
	}

	sig := lifecycle.WaitSignal()
	slog.Info("received shutdown signal", "signal", sig.String(), "iface", *iface)
}

func ebpfEdgeAttachAllowed() bool {
	redisAddr := edge.FirstRedisAddr()
	if redisAddr == "" {
		slog.Warn("REDIS_ADDRS unset; ebpf_xdp_edge entitlement check skipped")
		return true
	}

	rdb := redis.NewClient(netaddr.RedisClientOptions(redisAddr, os.Getenv("REDIS_PASS")))
	defer func() { _ = rdb.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	return edge.EbpfEdgeLicensed(ctx, rdb)
}
