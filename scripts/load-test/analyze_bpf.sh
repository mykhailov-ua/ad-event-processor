#!/usr/bin/env bash
# Generate bpf-report.md from BPF session artifacts.
# Usage: analyze_bpf.sh <load-test-out-dir>
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"

OUT="${1:?output dir required}"
BPF_DIR="$OUT/bpf"
SUMMARY="$BPF_DIR/maps/summary.json"
REPORT="$OUT/bpf-report.md"

if [[ ! -f "$SUMMARY" ]]; then
	printf 'analyze-bpf: no %s — skipping\n' "$SUMMARY" >&2
	exit 0
fi

python3 - "$OUT" "$SUMMARY" "$REPORT" <<'PY'
import json, os, sys
from datetime import datetime

out_dir, summary_path, report_path = sys.argv[1:4]
bpf_dir = os.path.join(out_dir, "bpf")

with open(summary_path) as f:
    data = json.load(f)

duration = data.get("duration_sec", 0)
pid_stats = data.get("pid_stats", [])
proc_samples = data.get("proc_samples", [])
cgroup_samples = data.get("cgroup_samples", [])
hot_syscalls = data.get("hot_syscalls", [])
syscalls = data.get("syscalls", [])
network = data.get("network", [])

def role_group(stats, role):
    return [s for s in stats if s.get("role") == role]

k6_stats = role_group(pid_stats, "k6")
tracker_stats = role_group(pid_stats, "tracker")
total_oncpu = sum(s.get("oncpu_ns", 0) for s in pid_stats)
k6_oncpu = sum(s.get("oncpu_ns", 0) for s in k6_stats)
k6_pct = (k6_oncpu / total_oncpu * 100) if total_oncpu else 0

lines = []
lines.append("# eSPX BPF Load-Test Report")
lines.append("")
lines.append(f"Generated: {datetime.utcnow().isoformat()}Z")
lines.append(f"Session dir: `{bpf_dir}`")
lines.append(f"Duration: {duration:.1f}s")
lines.append("")

lines.append("## Load generator overhead (k6)")
lines.append("")
lines.append("Honest capacity note: k6 context switches and on-CPU time are included so host saturation is visible separately from tracker work.")
lines.append("")
if k6_stats:
    lines.append("| k6 process | ctx/s | voluntary | involuntary | on-CPU % | RSS delta |")
    lines.append("|------------|-------|-----------|-------------|----------|-----------|")
    mem_start = os.path.join(bpf_dir, "mem-start.json")
    mem_end = os.path.join(bpf_dir, "mem-end.json")
    rss_start = {}
    rss_end = {}
    for path, store in ((mem_start, rss_start), (mem_end, rss_end)):
        if os.path.isfile(path):
            with open(path) as mf:
                mj = json.load(mf)
            for p in mj.get("processes", []):
                store[p["pid"]] = p.get("vm_rss_kb", 0)
    for s in sorted(k6_stats, key=lambda x: -x.get("oncpu_ns", 0)):
        pid = s["pid"]
        delta = rss_end.get(pid, 0) - rss_start.get(pid, 0)
        lines.append(
            f"| {s.get('name', pid)} | {s.get('ctx_switch_per_sec', 0):.0f} | "
            f"{s.get('voluntary_ctx', 0)} | {s.get('involuntary_ctx', 0)} | "
            f"{s.get('oncpu_pct', 0):.1f}% | {delta:+d} KB |"
        )
    lines.append("")
    lines.append(f"- **k6 share of tracked on-CPU time:** {k6_pct:.1f}%")
else:
    lines.append("_k6 not observed (set ESPX_BPF_TRACK_K6=1 during load; discovery runs automatically)._")
lines.append("")

lines.append("## Scheduler / context switches (services)")
lines.append("")
lines.append("| Process | Role | ctx/s | runqueue avg (µs) | runqueue p99 (µs) | on-CPU % | minor flt | major flt |")
lines.append("|---------|------|-------|-------------------|-------------------|----------|-----------|-----------|")
for s in sorted(pid_stats, key=lambda x: (x.get("role") == "k6", -x.get("ctx_switch_per_sec", 0))):
    if s.get("role") == "k6":
        continue
    lines.append(
        f"| {s.get('name', s['pid'])} | {s.get('role')} | {s.get('ctx_switch_per_sec', 0):.0f} | "
        f"{s.get('runqueue_avg_us', 0):.1f} | {s.get('runqueue_p99_us', 0):.1f} | "
        f"{s.get('oncpu_pct', 0):.1f} | {s.get('minor_faults', 0)} | {s.get('major_faults', 0)} |"
    )
lines.append("")

if cgroup_samples:
    lines.append("## Cgroup limits (CPU throttle & memory)")
    lines.append("")
    lines.append("From cgroup v2 `cpu.stat`, `memory.current`, `memory.events` sampled every 2s.")
    lines.append("")
    lines.append("| Container | Role | peak RAM (MiB) | peak anon (MiB) | throttle % | throttled (ms) | mem max events | IO read (MiB) | IO write (MiB) |")
    lines.append("|-----------|------|----------------|-----------------|------------|----------------|----------------|---------------|----------------|")
    for s in sorted(cgroup_samples, key=lambda x: -x.get("peak_memory_current", 0)):
        peak_mb = s.get("peak_memory_current", 0) / (1024 * 1024)
        anon_mb = s.get("peak_memory_anon", 0) / (1024 * 1024)
        thr_ms = s.get("total_cpu_throttled_usec", 0) / 1000
        r_mb = s.get("io_read_bytes", 0) / (1024 * 1024)
        w_mb = s.get("io_write_bytes", 0) / (1024 * 1024)
        lines.append(
            f"| {s.get('name', s['pid'])} | {s.get('role')} | {peak_mb:.0f} | {anon_mb:.0f} | "
            f"{s.get('throttle_pct', 0):.1f} | {thr_ms:.0f} | {s.get('memory_max_events', 0)} | "
            f"{r_mb:.1f} | {w_mb:.1f} |"
        )
    lines.append("")

if hot_syscalls:
    lines.append("## Hot syscalls (gnet / Redis path)")
    lines.append("")
    lines.append("Always traced: `epoll_wait`, `read`, `write`, `connect`, `sendto`, `recvfrom`, `futex`.")
    lines.append("")
    lines.append("| Role | syscall | count | avg (µs) | p99 (µs) | max (µs) |")
    lines.append("|------|---------|-------|----------|----------|----------|")
    for s in sorted(hot_syscalls, key=lambda x: (-1 if x.get("role") == "tracker" else 0, -x.get("p99_us", 0))):
        lines.append(
            f"| {s.get('role')} | {s.get('syscall')} | {s.get('count')} | "
            f"{s.get('avg_us', 0):.1f} | {s.get('p99_us', 0):.1f} | {s.get('max_us', 0):.1f} |"
        )
    lines.append("")

lines.append("## File descriptors & sockets")
lines.append("")
lines.append("Open FD counts come from `/proc/pid/fd` sampling every 2s; syscall counters (`openat`, `socket`, `accept`, `close`) are from BPF.")
lines.append("")
if proc_samples:
    lines.append("| Process | Role | peak FDs | peak sockets | FD Δ | open/s | close/s | socket() | accept() |")
    lines.append("|---------|------|----------|--------------|------|--------|---------|----------|----------|")
    for s in sorted(proc_samples, key=lambda x: -x.get("peak_open_fds", 0)):
        lines.append(
            f"| {s.get('name', s['pid'])} | {s.get('role')} | {s.get('peak_open_fds', 0)} | "
            f"{s.get('peak_socket_fds', 0)} | {s.get('fd_delta', 0):+d} | "
            f"{s.get('fd_open_per_sec', 0):.1f} | {s.get('fd_close_per_sec', 0):.1f} | "
            f"{s.get('socket_open', 0)} | {s.get('socket_accept', 0)} |"
        )
    lines.append("")
elif pid_stats:
    lines.append("| Process | Role | fd open/s | fd close/s | socket() | accept() | net FD est |")
    lines.append("|---------|------|-----------|------------|----------|----------|------------|")
    for s in sorted(pid_stats, key=lambda x: -x.get("fd_open_per_sec", 0)):
        lines.append(
            f"| {s.get('name', s['pid'])} | {s.get('role')} | {s.get('fd_open_per_sec', 0):.1f} | "
            f"{s.get('fd_close_per_sec', 0):.1f} | {s.get('socket_open', 0)} | {s.get('socket_accept', 0)} | "
            f"{s.get('net_fd_estimate', 0):+d} |"
        )
    lines.append("")

lines.append("## OS threads")
lines.append("")
lines.append("Thread count from `/proc/pid/status`; fork/exit events from `sched_process_{fork,exit}` tracepoints (per process TGID).")
lines.append("")
if proc_samples:
    lines.append("| Process | Role | peak threads | thread Δ | fork events | exit events |")
    lines.append("|---------|------|--------------|----------|-------------|-------------|")
    for s in sorted(proc_samples, key=lambda x: -x.get("peak_threads", 0)):
        lines.append(
            f"| {s.get('name', s['pid'])} | {s.get('role')} | {s.get('peak_threads', 0)} | "
            f"{s.get('thread_delta', 0):+d} | {s.get('thread_fork', 0)} | {s.get('thread_exit', 0)} |"
        )
    lines.append("")
elif pid_stats:
    lines.append("| Process | Role | fork events | exit events |")
    lines.append("|---------|------|-------------|-------------|")
    for s in sorted(pid_stats, key=lambda x: -x.get("thread_fork", 0)):
        if s.get("thread_fork", 0) == 0 and s.get("thread_exit", 0) == 0:
            continue
        lines.append(
            f"| {s.get('name', s['pid'])} | {s.get('role')} | {s.get('thread_fork', 0)} | {s.get('thread_exit', 0)} |"
        )
    lines.append("")

lines.append("## Syscalls (wall time)")
lines.append("")
lines.append("| Role | syscall | count | avg (µs) | p99 (µs) | max (µs) | wall % |")
lines.append("|------|---------|-------|----------|----------|----------|--------|")
for s in sorted(syscalls, key=lambda x: -x.get("wall_pct", 0))[:40]:
    lines.append(
        f"| {s.get('role')} | {s.get('syscall')} | {s.get('count')} | "
        f"{s.get('avg_us', 0):.1f} | {s.get('p99_us', 0):.1f} | {s.get('max_us', 0):.1f} | {s.get('wall_pct', 0):.1f} |"
    )
lines.append("")

events_path = os.path.join(bpf_dir, "events.ndjson")
if os.path.isfile(events_path):
    slow = []
    with open(events_path) as ef:
        for line in ef:
            line = line.strip()
            if not line:
                continue
            try:
                slow.append(json.loads(line))
            except json.JSONDecodeError:
                pass
    slow.sort(key=lambda x: -x.get("duration_us", 0))
    lines.append("## Slow syscall events")
    lines.append("")
    if slow:
        lines.append("| role | syscall | duration (µs) | pid |")
        lines.append("|------|---------|---------------|-----|")
        for e in slow[:25]:
            lines.append(
                f"| {e.get('role')} | {e.get('syscall_name')} | {e.get('duration_us')} | {e.get('pid')} |"
            )
    else:
        lines.append("_No slow events above threshold._")
    lines.append("")

if network:
    lines.append("## Network (TCP retrans)")
    lines.append("")
    lines.append("| process | role | retrans |")
    lines.append("|---------|------|---------|")
    for n in sorted(network, key=lambda x: -x.get("retrans", 0)):
        if n.get("retrans", 0) == 0:
            continue
        lines.append(f"| {n.get('name')} | {n.get('role')} | {n.get('retrans')} |")
    lines.append("")

lines.append("## Interpretation")
lines.append("")
lines.append("1. **k6 on-CPU > 15%** — load generator competes with tracker; lower RATE or run k6 on another host.")
lines.append("2. **tracker epoll_wait / read wall %** — gnet poll vs Redis RTT; compare with Prometheus redis_lua p99.")
lines.append("3. **involuntary ctx >> voluntary** — CPU oversubscription (GOMAXPROCS, compose CPU limits).")
lines.append("4. **cpu throttle % > 5%** — cgroup CPU limit is biting; raise compose cpus or lower RATE.")
lines.append("5. **memory max events > 0** — container hit memory.max; risk of OOM kill.")
lines.append("6. **tracker epoll_wait p99 high** — poll wait dominates; check connection count and Redis RTT.")
lines.append("7. **peak FDs growing / high open rate** — connection leak or missing close; compare peak sockets with Redis pool size.")
lines.append("8. **thread_fork >> thread_exit or peak threads climbing** — goroutine/thread pool growth; check GOMAXPROCS and gnet worker count.")
lines.append("9. **retrans > 0** — kernel TCP retry; check Redis/network chaos or laptop Wi-Fi.")
lines.append("")

with open(report_path, "w") as rf:
    rf.write("\n".join(lines))

print(f"analyze-bpf: wrote {report_path}")
PY
