// Ringbuf slow-event drain to events.ndjson and optional OTEL export.
//
// Role:
//   - drainRingbufRecords appends JSON rows per DecodeSlowEvent to sessionDir/events.ndjson.
//   - Enriches syscall_id with static x86_64 name table; duration_us = dur_ns / 1000.
//
// events.ndjson contract:
//   - OpenFile CREATE|WRONLY|APPEND; one JSON object per line (json.Encoder.Encode, no array wrapper).
//   - Fields: ts_ns, pid, role, syscall_id, syscall_name, duration_us, kind, campaign_slot, marker_id, marker_name.
//   - Append-only for load-report / bpf-report; not a generational snapshot (file grows for session TTL).
//   - Encoder errors ignored per row (fail-open row drop); OpenFile failure logs warn and returns (no retry).
//
// Topology:
//   - Fed by BPF slow_events map; optional otelLogExporter batches to OTLP /v1/logs HTTP.
//
// Invariants:
//   - ctx.Done() checked only between rd.Read calls; Read blocks in kernel poll until reader close.
//   - Shutdown: probeRun.stop cancels ctx, closeRingbufReader unblocks rd.Read, then ringWG.Wait.
//   - releaseRingbufReader is idempotent; stop and drain defer must not double-close the same reader.
//   - OTEL emit uses non-blocking select; queue full drops event (fail-open, debug log only).
//
// Verify:
//
//	go test ./cmd/bpf-collector/ -short -run 'TestProbeRun_stop_.*Ringbuf' -count=1
//	wc -l var/load-test/<session>/events.ndjson
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cilium/ebpf/ringbuf"
)

func (r *probeRun) registerRingbufReader(rd *ringbuf.Reader) {
	r.ringMu.Lock()
	r.ringRD = rd
	r.ringMu.Unlock()
}

func (r *probeRun) closeRingbufReader() {
	r.ringMu.Lock()
	rd := r.ringRD
	r.ringRD = nil
	r.ringMu.Unlock()
	if rd != nil {
		_ = rd.Close()
	}
}

func (r *probeRun) releaseRingbufReader(rd *ringbuf.Reader) {
	r.ringMu.Lock()
	if r.ringRD == rd {
		r.ringRD = nil
		_ = rd.Close()
	}
	r.ringMu.Unlock()
}

func drainRingbufRecords(ctx context.Context, rd *ringbuf.Reader, sessionDir string, otel *otelLogExporter) {
	outPath := filepath.Join(sessionDir, "events.ndjson")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Warn("events file", "error", err)
		return
	}
	defer func() { _ = f.Close() }()
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
		e := DecodeSlowEvent(rec.RawSample)
		row := map[string]any{
			"ts_ns":         e.TSNs,
			"pid":           e.PID,
			"role":          roleName(e.Role),
			"syscall_id":    e.SyscallID,
			"syscall_name":  syscallName(int(e.SyscallID)),
			"duration_us":   e.DurNs / 1000,
			"kind":          e.Kind,
			"campaign_slot": e.CampaignSlot,
			"marker_id":     e.MarkerID,
			"marker_name":   markerName(e.MarkerID),
		}
		_ = enc.Encode(row)
		if otel != nil {
			otel.emit(otelLogRecord(row))
		}
	}
}

func syscallName(nr int) string {
	names := map[int]string{
		0: "read", 1: "write", 3: "close", 7: "poll", 9: "mmap", 16: "ioctl",
		19: "writev", 41: "socket", 42: "connect", 43: "accept", 44: "sendto", 45: "recvfrom",
		74: "fsync", 75: "fdatasync",
		57: "clone", 202: "futex", 228: "clock_gettime", 230: "exit_group",
		232: "epoll_wait", 233: "epoll_ctl", 257: "openat", 291: "epoll_create1",
		318: "getrandom",
	}
	if n, ok := names[nr]; ok {
		return n
	}
	return fmt.Sprintf("sys_%d", nr)
}

func markerName(id uint32) string {
	switch id {
	case 1:
		return "process_track"
	case 3:
		return "filter_check"
	default:
		return fmt.Sprintf("marker_%d", id)
	}
}
