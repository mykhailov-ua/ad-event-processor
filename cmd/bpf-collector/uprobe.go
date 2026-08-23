package main

import (
	"debug/elf"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/naming"
	"github.com/cilium/ebpf/link"
)

type uprobeSpec struct {
	symbol string
	enter  bool
	cookie uint64
}

var trackerUprobeSpecs = []uprobeSpec{
	{symbol: "github.com/bidshard/ad-event-processor/internal/ingestion/traceprobe.ProcessTrackEnter", enter: true, cookie: 1},
	{symbol: "github.com/bidshard/ad-event-processor/internal/ingestion/traceprobe.ProcessTrackExit", enter: false, cookie: 2},
	{symbol: "github.com/bidshard/ad-event-processor/internal/ingestion/traceprobe.FilterCheckEnter", enter: true, cookie: 3},
	{symbol: "github.com/bidshard/ad-event-processor/internal/ingestion/traceprobe.FilterCheckExit", enter: false, cookie: 4},
}

func (r *probeRun) attachUprobes() {
	bin := r.trackerBinary
	if bin == "" {
		bin = r.findTrackerBinary()
	}
	if bin == "" {
		slog.Info("uprobes skipped: tracker binary not found (build tracker with -tags " + naming.BPFTraceBuildTag() + ")")
		return
	}
	if r.coll == nil {
		return
	}

	attached := 0
	ex, err := link.OpenExecutable(bin)
	if err != nil {
		slog.Warn("uprobe open executable", "binary", bin, "error", err)
		return
	}
	for _, spec := range trackerUprobeSpecs {
		prog := r.coll.Progs.TraceEnter
		if !spec.enter {
			prog = r.coll.Progs.TraceExit
		}
		if prog == nil {
			continue
		}
		sym := resolveGoSymbol(bin, spec.symbol)
		if sym == "" {
			slog.Debug("uprobe symbol missing", "want", spec.symbol)
			continue
		}
		l, err := ex.Uprobe(sym, prog, &link.UprobeOptions{Cookie: spec.cookie})
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

func (r *probeRun) findTrackerBinary() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tracked {
		if t.Role != roleTracker || t.PID == 0 {
			continue
		}
		exe, err := os.Readlink(filepath.Join(string(filepath.Separator), "proc", strconv.FormatUint(uint64(t.PID), 10), "exe"))
		if err != nil {
			continue
		}
		return exe
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
