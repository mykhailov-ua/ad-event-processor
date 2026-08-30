// Loadgen PID discovery via /proc comm and cmdline scan.
//
// Role:
//   - discoverLoop ticks -discover-interval (default 2s); scans /proc for loadgen processes.
//   - Matches comm against -loadgen-comms or AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM; also cmd/loadgen path.
//   - trackLoadgenChildren attaches threads sharing Tgid with discovered leader PID.
//
// Topology:
//   - Complements static targets.json; does not remove exited PIDs from BPF maps.
//
// Invariants:
//   - Default comm list ["loadgen"] when env/flag empty.
//   - ReadDir /proc failure skips tick silently.
//
// Verify:
//   go test ./cmd/bpf-collector/... -short -run Discover -count=1
package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (r *probeRun) discoverLoop(ctx context.Context) {
	ticker := time.NewTicker(r.discoverTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.discoverLoadgenPIDs()
		}
	}
}

func (r *probeRun) discoverLoadgenPIDs() {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	comms := r.loadgenComms
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		pid := uint32(pid64)
		comm, err := os.ReadFile(filepath.Join(string(filepath.Separator), "proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(comm))
		cmdline := procCmdline(pid)
		if !loadgenProcessMatch(name, cmdline, comms) {
			continue
		}
		label := fmt.Sprintf("loadgen:%s:%d", name, pid)
		if err := r.trackPID(pid, roleLoadgen, label); err != nil {
			slog.Debug("loadgen track", "pid", pid, "comm", name, "error", err)
		}
		r.trackLoadgenChildren(pid, name)
	}
}

func loadgenCommMatch(comm string, allowed []string) bool {
	for _, a := range allowed {
		if comm == a {
			return true
		}
	}
	return false
}

func procCmdline(pid uint32) string {
	data, err := os.ReadFile(filepath.Join(string(filepath.Separator), "proc", strconv.FormatUint(uint64(pid), 10), "cmdline"))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(data), "\x00", " ")
}

func loadgenProcessMatch(comm, cmdline string, allowed []string) bool {
	if loadgenCommMatch(comm, allowed) {
		return true
	}
	if cmdline == "" {
		return false
	}
	if strings.Contains(cmdline, "cmd/loadgen") {
		return true
	}
	if strings.Contains(cmdline, "/loadgen ") || strings.HasSuffix(strings.TrimSpace(cmdline), "/loadgen") {
		return true
	}
	return false
}

func parseLoadgenComms(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"loadgen"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"loadgen"}
	}
	return out
}

func (r *probeRun) trackLoadgenChildren(leader uint32, comm string) {
	entries, err := os.ReadDir(filepath.Join(string(filepath.Separator), "proc"))
	if err != nil {
		return
	}
	for _, e := range entries {
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		pid := uint32(pid64)
		statusPath := filepath.Join(string(filepath.Separator), "proc", e.Name(), "status")
		tgid := procTGID(statusPath)
		if tgid != leader || pid == leader {
			continue
		}
		_ = r.trackPID(pid, roleLoadgen, fmt.Sprintf("loadgen:%s-thread:%d", comm, pid))
	}
}

func procTGID(statusPath string) uint32 {
	f, err := os.Open(statusPath)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "Tgid:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0
			}
			v, _ := strconv.ParseUint(fields[1], 10, 32)
			return uint32(v)
		}
	}
	if err := sc.Err(); err != nil {
		return 0
	}
	return 0
}
