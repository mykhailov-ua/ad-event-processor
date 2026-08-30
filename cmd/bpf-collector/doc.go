// Package main attaches load-test eBPF probes and dumps session BPF artifacts.
//
// Role:
//   - Require -session-dir (targets.json + output tree under var/load-test/<utc>/).
//   - Load deploy/dev/bpf/loadtest_probe.o; attach tracepoints (syscalls, sched, page fault, TCP retransmit).
//   - Optional tracker uprobes (traceprobe build tag); discover loadgen PIDs by /proc comm when -discover-loadgen.
//   - Drain slow-events ringbuf to events.ndjson; periodic or shutdown dump of bpf/maps/summary.json.
//   - Optional Prometheus /metrics and OTEL log export from probe session.
//
// Layout constants (session-dir inputs):
//   - targets.json (required): JSON sessionMeta with started_at, sample_rate, roles_wanted, targets[].
//   - Each targetEntry: pid, cgroup_id (optional), role (1=tracker, 2=nginx, 3=redis, 4=loadgen, 5=processor), name.
//
// Layout constants (session-dir outputs):
//   - session.json: meta with ended_at set on graceful stop.
//   - timeline.json: started_at, prometheus_url, roles_wanted, sample_rate, dump_interval_s.
//   - events.ndjson: one JSON object per slow ringbuf record (append-only).
//   - proc-samples.ndjson, cgroup-samples.ndjson: 2s ticker samples while running.
//   - mem-start.json, mem-end.json: /proc status snapshots for tracked PIDs.
//   - bpf/maps/summary.json: aggregated PID stats, syscalls, network, markers, hardware_perf (linux).
//
// Topology:
//   - Runs alongside loadgen during AD_EVENT_PROCESSOR_BPF_PROBE=1 sessions.
//   - Requires CAP_BPF (tracepoint/kprobe/uprobe attach) and memlock (rlimit.RemoveMemlock at startup).
//   - Not production edge-xdp compliance maps; dev-host load-test instrumentation only.
//
// Env defaults:
//   - AD_EVENT_PROCESSOR_BPF_PROBE, AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE, AD_EVENT_PROCESSOR_BPF_SLOW_US, AD_EVENT_PROCESSOR_BPF_TARGETS (load-test-bpf.mdc).
//
// Ringbuf drain:
//   - Goroutine reads slow_events BPF_MAP_TYPE_RINGBUF via cilium/ebpf/ringbuf.Reader.
//   - drainRingbufRecords loops: check ctx.Done() between reads; rd.Read blocks until reader close.
//   - Shutdown: cancel ctx, closeRingbufReader (unblocks Read), ringWG.Wait, then sampleWG and coll.Close.
//   - OTEL side channel capacity 256; emit drops on full (non-blocking select default).
//
// Defaults and limits:
//   - -dump-interval 0 (default): no periodic dump; dumpMaps runs once in probeRun.stop().
//   - -dump-interval > 0: dumpLoop ticker writes bpf/maps/summary.json each interval (overwrites).
//   - slow-us default 10000 us (10 ms): syscalls longer enqueue slow_events ringbuf record.
//   - sample-rate default 1 (every Nth syscall in kernel); use 10 on laptops (load-test-bpf.mdc).
//   - discover-interval default 2s: /proc scan for loadgen comm/cmdline.
//   - refresh-targets default 0 (disabled); when set, re-runs scripts/test/bpf_resolve_targets.sh.
//   - sessionDuration floor 1s when ended_at <= started_at (avoids div-by-zero in rates).
//   - OTEL batch flush: 32 records or 2s ticker; HTTP client timeout 5s per export POST.
//
// Invariants:
//   - session-dir required; probe start failure exits 1; missing -session-dir exits 2.
//   - On SIGTERM: cancel workers, close ringbuf reader before ringWG.Wait, dump maps, mem-end snapshot, mark session ended.
//   - Tracepoint attach failures log warn and continue (partial probe set allowed).
//   - SysEnter/SysExit required in .o; Load fails hard if missing.
//
// Forbidden:
//   - Citing BPF syscall counts or oncpu_pct as /track production p99 (use Prometheus ad_http_request_duration_seconds).
//   - Running on tracker prod hosts without load-test session isolation.
//   - Claiming zero-alloc on tracker hot path from BPF session metrics.
//
// Verify:
// make bpf-dev
// AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/load/malformed.sh business
// make bpf-session-start && make bpf-session-stop
// go test ./cmd/bpf-collector/... -short -run 'TestProbeRun_stop_.*Ringbuf' -count=1
// go test ./cmd/bpf-collector/... -short -count=1
// go run ./cmd/load-report all var/load-test/<session>/
package main
