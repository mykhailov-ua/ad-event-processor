// XDP map pinning and attach helpers for cmd/edge-xdp.
//
// Memory Model Rules:
//
// pinMaps lifecycle:
//   - Unpins stale bpf fs links (os.Remove) before Pin to avoid EEXIST on restart.
//   - All 16 maps must be non-nil after LoadEdgeObjectsLenient or pinMaps errors out.
//   - prog_array pinned with maps so edge-bpf-sync era consumers see stable paths; only edge-xdp
//     writes prog_array slot 0 (wireProgArray before pinMaps in main init order).
//
// attachXDPWithFallback:
//   - Licensed attach only (main.ebpfEdgeAttachAllowed). Modes: offload->native->generic,
//     native->generic, or generic-only per XDP_MODE env.
//   - Link lifetime tied to process; map pins survive for edge-bpf-sync LoadPinned* opens.
//
// Cross-layer note:
//   - Pinned blocklist_host_* maps use LRU_HASH; explicit Delete on remove comes from edge-bpf-sync
//     BlocklistStore, not from this file. XDP attach does not populate deny data.
package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"ad-event-processor/internal/edge"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// pinMaps pins 15 kernel maps under pinDir for edge-bpf-sync to populate via bpf fs.
func pinMaps(objs *edge.EdgeObjects, pinDir string) error {
	pins := map[string]*ebpf.Map{
		"blocklist_v4":            objs.BlocklistV4,
		"blocklist_v6":            objs.BlocklistV6,
		"blocklist_host_v4":       objs.BlocklistHostV4,
		"blocklist_host_v6":       objs.BlocklistHostV6,
		"allow_v4":                objs.AllowV4,
		"allow_v6":                objs.AllowV6,
		"syn_ratelimit_v4":        objs.SynRatelimitV4,
		"syn_subnet_ratelimit_v4": objs.SynSubnetRatelimitV4,
		"ratelimit_v4":            objs.RatelimitV4,
		"rst_ratelimit_v4":        objs.RstRatelimitV4,
		"global_syn":              objs.GlobalSyn,
		"stats":                   objs.Stats,
		"config":                  objs.Config,
		"violations":              objs.Violations,
		"fingerprints":            objs.Fingerprints,
		"prog_array":              objs.ProgArray,
	}
	for name, m := range pins {
		if m == nil {
			return fmt.Errorf("map %s not loaded", name)
		}
		path := filepath.Join(pinDir, name)
		_ = os.Remove(path)
		if err := m.Pin(path); err != nil {
			return fmt.Errorf("pin %s: %w", name, err)
		}
	}
	return nil
}

func attachXDP(iface string, prog *ebpf.Program, mode string) (link.Link, error) {
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}
	switch mode {
	case "generic":
		return link.AttachXDP(link.XDPOptions{
			Program:   prog,
			Interface: ifaceObj.Index,
			Flags:     link.XDPGenericMode,
		})
	case "native":
		return link.AttachXDP(link.XDPOptions{
			Program:   prog,
			Interface: ifaceObj.Index,
			Flags:     link.XDPDriverMode,
		})
	case "offload":
		return link.AttachXDP(link.XDPOptions{
			Program:   prog,
			Interface: ifaceObj.Index,
			Flags:     link.XDPOffloadMode,
		})
	default:
		return nil, fmt.Errorf("unknown XDP_MODE %q", mode)
	}
}

// xdpAttachModes returns fallback chain: offload->native->generic, native->generic, or generic only.
func xdpAttachModes(mode string) []string {
	switch mode {
	case "offload":
		return []string{"offload", "native", "generic"}
	case "native":
		return []string{"native", "generic"}
	case "generic":
		return []string{"generic"}
	default:
		return []string{mode}
	}
}

func attachXDPWithFallback(iface string, prog *ebpf.Program, mode string) (link.Link, string, error) {
	modes := xdpAttachModes(mode)
	var lastErr error
	for i, m := range modes {
		lnk, err := attachXDP(iface, prog, m)
		if err == nil {
			if i > 0 {
				slog.Warn("xdp attach fell back to slower mode", "requested", mode, "attached", m)
			}
			return lnk, m, nil
		}
		lastErr = err
		if i < len(modes)-1 {
			slog.Warn("xdp attach failed, trying fallback", "mode", m, "error", err)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no attach modes to try for %q", mode)
	}
	return nil, mode, lastErr
}
