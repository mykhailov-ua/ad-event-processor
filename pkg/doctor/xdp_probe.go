package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"ad-event-processor/internal/edge"
)

const (
	edgeXDPUnitName     = "ad-event-processor-edge-xdp.service"
	edgeBPFSyncUnitName = "ad-event-processor-edge-bpf-sync.service"
	statsFreshnessLimit = 2 * time.Minute
)

var (
	btfPathForProbe          = "/sys/kernel/btf/vmlinux"
	blocklistMapPathForProbe = edge.DefaultBlocklistMapPath
	systemdUnitActiveFn      = systemdUnitActive
)

type EdgeXDPProbe struct {
	ConfigEnabled bool
	Deps          ProbeDeps
	StatsReader   func(context.Context) (edge.Snapshot, error)
}

func (EdgeXDPProbe) Name() string { return "edge_xdp" }

func (p EdgeXDPProbe) Run(ctx context.Context) Result {
	start := time.Now()
	latency := func() int64 { return time.Since(start).Milliseconds() }

	if !p.ConfigEnabled {
		return Result{
			Name:    "edge_xdp",
			Status:  StatusSkip,
			Detail:  "edge_xdp disabled in platform settings",
			Latency: latency(),
		}
	}

	var fails, warns []string

	if ok, detail := ebpfEntitled(ctx, p.Deps); !ok {
		fails = append(fails, detail)
	}

	if _, err := os.Stat(btfPathForProbe); err != nil {
		fails = append(fails, "BTF vmlinux missing at "+btfPathForProbe)
	}

	if _, err := os.Stat(blocklistMapPathForProbe); err != nil {
		fails = append(fails, "blocklist_v4 map not pinned at "+blocklistMapPathForProbe)
	}

	switch unitsActive, detail := edgeSystemdStatus(); {
	case unitsActive:

	case detail == "systemctl unavailable":
		warns = append(warns, "systemd units not verified (systemctl unavailable)")
	default:
		fails = append(fails, detail)
	}

	if p.StatsReader != nil {
		snap, err := p.StatsReader(ctx)
		switch {
		case err != nil:
			warns = append(warns, "xdp stats snapshot missing in Redis")
		case snap.UpdatedAt.IsZero():
			warns = append(warns, "xdp stats snapshot has no updated_at")
		case time.Since(snap.UpdatedAt) > statsFreshnessLimit:
			warns = append(warns, fmt.Sprintf("xdp stats snapshot stale (%s old)", time.Since(snap.UpdatedAt).Round(time.Second)))
		}
	}

	if len(fails) > 0 {
		return Result{
			Name:    "edge_xdp",
			Status:  StatusFail,
			Detail:  strings.Join(append(fails, warns...), "; "),
			Latency: latency(),
		}
	}
	if len(warns) > 0 {
		return Result{
			Name:    "edge_xdp",
			Status:  StatusWarn,
			Detail:  strings.Join(warns, "; "),
			Latency: latency(),
		}
	}
	return Result{
		Name:    "edge_xdp",
		Status:  StatusPass,
		Detail:  "BTF present, maps pinned, edge systemd units active, stats snapshot fresh",
		Latency: latency(),
	}
}

func ebpfEntitled(ctx context.Context, deps ProbeDeps) (entitled bool, reason string) {
	if deps.Redis == nil {
		return true, ""
	}
	clients, err := deps.Redis(ctx)
	if err != nil || len(clients) == 0 || clients[0] == nil {
		return true, ""
	}
	enabled, err := clients[0].HGet(ctx, "entitlement:deployment", "ebpf_xdp_edge").Int()
	if err != nil {
		return true, ""
	}
	if enabled != 1 {
		return false, "ebpf_xdp_edge entitlement disabled in Redis"
	}
	return true, ""
}

func edgeSystemdStatus() (active bool, detail string) {
	if systemdUnitActiveFn == nil {
		return false, "systemd probe not configured"
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false, "systemctl unavailable"
	}
	for _, unit := range []string{edgeXDPUnitName, edgeBPFSyncUnitName} {
		ok, err := systemdUnitActiveFn(unit)
		if err != nil {
			return false, fmt.Sprintf("%s check failed: %v", unit, err)
		}
		if !ok {
			return false, fmt.Sprintf("%s not active", unit)
		}
	}
	return true, ""
}

func systemdUnitActive(unit string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", unit)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() != 0 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
