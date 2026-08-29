package main

import (
	"debug/elf"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ad-event-processor/pkg/naming"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

type uprobeAttach struct {
	symbol string
	prog   *ebpf.Program
}

func (r *probeRun) attachUprobes() {
	bin := r.resolveTrackerBinary()
	if bin == "" {
		slog.Info("uprobes skipped: tracker binary not found (build tracker with -tags " + naming.BPFTraceBuildTag() + ")")
		return
	}
	if r.coll == nil {
		return
	}

	progs := []uprobeAttach{
		{"ad-event-processor/internal/ingest/traceprobe.ProcessTrackEnter", r.coll.Progs.MarkProcessTrackEnter},
		{"ad-event-processor/internal/ingest/traceprobe.ProcessTrackExit", r.coll.Progs.MarkProcessTrackExit},
		{"ad-event-processor/internal/filter.FilterCheckEnter", r.coll.Progs.MarkFilterCheckEnter},
		{"ad-event-processor/internal/filter.FilterCheckExit", r.coll.Progs.MarkFilterCheckExit},
	}

	attached := 0
	ex, err := link.OpenExecutable(bin)
	if err != nil {
		slog.Warn("uprobe open executable", "binary", bin, "error", err)
		return
	}
	for _, spec := range progs {
		if spec.prog == nil {
			continue
		}
		sym := resolveGoSymbol(bin, spec.symbol)
		if sym == "" {
			slog.Debug("uprobe symbol missing", "want", spec.symbol)
			continue
		}
		l, err := ex.Uprobe(sym, spec.prog, nil)
		if err != nil {
			slog.Warn("uprobe attach skipped", "symbol", sym, "error", err)
			continue
		}
		r.links = append(r.links, l)
		attached++
	}
	if attached == 0 {
		slog.Info("uprobes not attached", "binary", bin, "hint", "go build -tags "+naming.BPFTraceBuildTag()+" -o bin/tracker ./cmd/tracker")
		return
	}
	slog.Info("uprobes attached", "binary", bin, "count", attached)
}

func chooseTrackerBinary(procExePath, flagBinary string) string {
	if procExePath != "" {
		return procExePath
	}
	return flagBinary
}

func (r *probeRun) resolveTrackerBinary() string {
	if r.trackerBinary != "" {
		if _, statErr := os.Stat(r.trackerBinary); statErr == nil {
			return r.trackerBinary
		} else {
			slog.Warn("uprobe flagged binary missing", "binary", r.trackerBinary, "error", statErr)
		}
	}
	chosen := chooseTrackerBinary(r.findTrackerProcExe(), "")
	if chosen == "" {
		return ""
	}
	if _, err := os.Stat(chosen); err != nil {
		slog.Warn("uprobe binary missing", "binary", chosen, "error", err)
		return ""
	}
	return chosen
}

func (r *probeRun) findTrackerProcExe() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tracked {
		if t.Role != roleTracker || t.PID == 0 {
			continue
		}
		pid := strconv.FormatUint(uint64(t.PID), 10)
		procExe := filepath.Join("/proc", pid, "exe")
		if _, err := os.Stat(procExe); err == nil {
			return procExe
		}
		procRoot := filepath.Join("/proc", pid, "root", "tracker")
		if _, err := os.Stat(procRoot); err == nil {
			return procRoot
		}
	}
	return ""
}

func resolveGoSymbol(binaryPath, want string) string {
	f, err := elf.Open(binaryPath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	syms, err := f.Symbols()
	if err != nil {
		syms, err = f.DynamicSymbols()
		if err != nil {
			return ""
		}
	}
	for _, s := range syms {
		if s.Name == want {
			return s.Name
		}
	}
	short := want[strings.LastIndex(want, "/")+1:]
	for _, s := range syms {
		if strings.HasSuffix(s.Name, short) && strings.Contains(s.Name, "traceprobe") {
			return s.Name
		}
	}
	return ""
}
