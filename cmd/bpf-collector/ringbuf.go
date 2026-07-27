package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"espx/cmd/bpf-collector/bpfprobe"

	"github.com/cilium/ebpf/ringbuf"
)

func (r *probeRun) discoverLoop(ctx context.Context) {
	ticker := time.NewTicker(r.discoverTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.discoverK6PIDs()
		}
	}
}

func (r *probeRun) discoverK6PIDs() {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		pid := uint32(pid64)
		comm, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(comm))
		if name != "k6" {
			continue
		}
		label := fmt.Sprintf("k6:%d", pid)
		if err := r.trackPID(pid, roleK6, label); err != nil {
			slog.Debug("k6 track", "pid", pid, "error", err)
		}
		r.trackK6Children(pid)
	}
}

func (r *probeRun) trackK6Children(leader uint32) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	for _, e := range entries {
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		pid := uint32(pid64)
		statusPath := filepath.Join("/proc", e.Name(), "status")
		tgid := procTGID(statusPath)
		if tgid != leader || pid == leader {
			continue
		}
		_ = r.trackPID(pid, roleK6, fmt.Sprintf("k6-thread:%d", pid))
	}
}

func procTGID(statusPath string) uint32 {
	f, err := os.Open(statusPath)
	if err != nil {
		return 0
	}
	defer f.Close()
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
	return 0
}

func drainRingbufRecords(ctx context.Context, rd *ringbuf.Reader, sessionDir string) {
	outPath := filepath.Join(sessionDir, "events.ndjson")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Warn("events file", "error", err)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		rec, err := rd.Read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		tsNs, pid, syscallID, durNs, role, kind := bpfprobe.DecodeSlowEvent(rec.RawSample)
		row := map[string]any{
			"ts_ns":        tsNs,
			"pid":          pid,
			"role":         roleName(role),
			"syscall_id":   syscallID,
			"syscall_name": syscallName(int(syscallID)),
			"duration_us":  durNs / 1000,
			"kind":         kind,
		}
		_ = enc.Encode(row)
	}
}

func syscallName(nr int) string {
	names := map[int]string{
		0: "read", 1: "write", 3: "close", 7: "poll", 9: "mmap", 16: "ioctl",
		41: "socket", 42: "connect", 43: "accept", 44: "sendto", 45: "recvfrom",
		57: "clone", 202: "futex", 228: "clock_gettime", 230: "exit_group",
		232: "epoll_wait", 233: "epoll_ctl", 257: "openat", 291: "epoll_create1",
		318: "getrandom",
	}
	if n, ok := names[nr]; ok {
		return n
	}
	return fmt.Sprintf("sys_%d", nr)
}
