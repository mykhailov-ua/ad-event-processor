// SPDX-License-Identifier: GPL-2.0
// Dev-only load-test probes: syscalls, scheduler context switches, TCP signals.
// Attach from cmd/bpf-collector during k6 runs; never loaded in production tracker.

#include <linux/bpf.h>
#include <linux/sched.h>
#include <linux/errno.h>

#include <bpf/bpf_core_read.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#include "common/espx_probe.h"
#include "common/espx_trace.h"

char LICENSE[] SEC("license") = "GPL";

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, __u8);
} target_pids SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct espx_config);
} config SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 8192);
	__type(key, __u64);
	__type(value, __u64);
} syscall_enter SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 4096);
	__type(key, struct espx_syscall_hist_key);
	__type(value, struct espx_hist);
} syscall_hist SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, struct espx_pid_stats);
} pid_stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, __u64);
} wakeup_ts SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 2048);
	__type(key, struct espx_net_key);
	__type(value, struct espx_net_stats);
} net_stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 20);
} slow_events SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, struct espx_hist);
} runqueue_hist SEC(".maps");

static __always_inline struct espx_config *probe_config(void)
{
	__u32 key = 0;

	return bpf_map_lookup_elem(&config, &key);
}

static __always_inline struct espx_pid_stats *pid_stats_mut(__u32 pid, __u8 role)
{
	struct espx_pid_stats *st;
	struct espx_pid_stats init = {};

	st = bpf_map_lookup_elem(&pid_stats, &pid);
	if (st)
		return st;
	init.role = role;
	bpf_map_update_elem(&pid_stats, &pid, &init, BPF_NOEXIST);
	return bpf_map_lookup_elem(&pid_stats, &pid);
}

SEC("tracepoint/raw_syscalls/sys_enter")
int espx_sys_enter(struct trace_event_raw_sys_enter *ctx)
{
	struct espx_config *cfg;
	__u64 pid_tgid;
	__u32 pid;
	__u64 ts;
	long syscall_id;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	syscall_id = ctx->id;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = pid_tgid >> 32;
	if (!espx_target_role(&target_pids, pid))
		return 0;

	if (!espx_should_sample(cfg) && !espx_is_hot_syscall(syscall_id))
		return 0;

	ts = bpf_ktime_get_ns();
	bpf_map_update_elem(&syscall_enter, &pid_tgid, &ts, BPF_ANY);
	return 0;
}

SEC("tracepoint/raw_syscalls/sys_exit")
int espx_sys_exit(struct trace_event_raw_sys_exit *ctx)
{
	struct espx_config *cfg;
	struct espx_syscall_hist_key hkey;
	struct espx_hist *hist;
	struct espx_slow_event ev;
	__u64 *enter_ts;
	__u64 pid_tgid;
	__u32 pid;
	__u64 now;
	__u64 delta;
	__u8 role;
	struct espx_pid_stats *st;
	long syscall_id;
	long syscall_ret;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	syscall_id = ctx->id;
	syscall_ret = ctx->ret;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = pid_tgid >> 32;
	role = espx_target_role(&target_pids, pid);
	if (!role)
		return 0;

	st = pid_stats_mut(pid, role);
	espx_account_fd_exit(st, syscall_id, syscall_ret);

	enter_ts = bpf_map_lookup_elem(&syscall_enter, &pid_tgid);
	if (!enter_ts)
		return 0;
	now = bpf_ktime_get_ns();
	delta = now - *enter_ts;
	bpf_map_delete_elem(&syscall_enter, &pid_tgid);

	hkey.pid = pid;
	hkey.syscall_id = syscall_id;
	hist = bpf_map_lookup_elem(&syscall_hist, &hkey);
	if (!hist) {
		struct espx_hist fresh = {};

		bpf_map_update_elem(&syscall_hist, &hkey, &fresh, BPF_NOEXIST);
		hist = bpf_map_lookup_elem(&syscall_hist, &hkey);
	}
	if (hist)
		espx_hist_record(hist, delta);

	if (cfg->slow_syscall_ns && delta >= cfg->slow_syscall_ns) {
		__builtin_memset(&ev, 0, sizeof(ev));
		ev.ts_ns = now;
		ev.pid = pid;
		ev.syscall_id = syscall_id;
		ev.duration_ns = delta;
		ev.role = role;
		ev.kind = 1;
		bpf_ringbuf_output(&slow_events, &ev, sizeof(ev), 0);
	}
	return 0;
}

SEC("tracepoint/sched/sched_wakeup")
int espx_sched_wakeup(struct trace_event_raw_sched_wakeup *ctx)
{
	struct espx_config *cfg;
	__u32 pid;
	__u64 ts;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid = ctx->pid;
	role = espx_target_role(&target_pids, pid);
	if (!role)
		return 0;

	ts = bpf_ktime_get_ns();
	bpf_map_update_elem(&wakeup_ts, &pid, &ts, BPF_ANY);
	return 0;
}

SEC("tracepoint/sched/sched_switch")
int espx_sched_switch(struct trace_event_raw_sched_switch *ctx)
{
	struct espx_config *cfg;
	struct espx_pid_stats *st;
	struct espx_hist *rq_hist;
	struct espx_hist rq_fresh = {};
	__u64 now;
	__u32 prev_pid;
	__u32 next_pid;
	__u8 prev_role;
	__u8 next_role;
	__u64 *wake;
	__u64 wait_ns;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	now = bpf_ktime_get_ns();
	prev_pid = ctx->prev_pid;
	next_pid = ctx->next_pid;
	prev_role = espx_target_role(&target_pids, prev_pid);
	next_role = espx_target_role(&target_pids, next_pid);

	if (prev_role) {
		st = pid_stats_mut(prev_pid, prev_role);
		if (st) {
			st->ctx_switch_out++;
			if (ctx->prev_state > 0)
				st->voluntary_ctx++;
			else
				st->involuntary_ctx++;
			if (st->last_oncpu_ns)
				st->oncpu_ns += now - st->last_oncpu_ns;
			st->last_oncpu_ns = 0;
		}
	}

	if (next_role) {
		st = pid_stats_mut(next_pid, next_role);
		if (st) {
			st->ctx_switch_in++;
			st->last_oncpu_ns = now;
			wake = bpf_map_lookup_elem(&wakeup_ts, &next_pid);
			if (wake && now > *wake) {
				wait_ns = now - *wake;
				st->runqueue_ns += wait_ns;
				st->runqueue_samples++;
				rq_hist = bpf_map_lookup_elem(&runqueue_hist, &next_pid);
				if (!rq_hist) {
					bpf_map_update_elem(&runqueue_hist, &next_pid, &rq_fresh, BPF_NOEXIST);
					rq_hist = bpf_map_lookup_elem(&runqueue_hist, &next_pid);
				}
				if (rq_hist)
					espx_hist_record(rq_hist, wait_ns);
			}
		}
	}
	return 0;
}

SEC("tracepoint/exceptions/page_fault_user")
int espx_page_fault_user(struct trace_event_raw_page_fault_user *ctx)
{
	struct espx_config *cfg;
	struct espx_pid_stats *st;
	__u32 pid;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid = bpf_get_current_pid_tgid() >> 32;
	role = espx_target_role(&target_pids, pid);
	if (!role)
		return 0;

	st = pid_stats_mut(pid, role);
	if (st)
		st->minor_faults++;
	return 0;
}

SEC("kprobe/tcp_retransmit_skb")
int espx_tcp_retransmit(struct pt_regs *ctx)
{
	struct espx_config *cfg;
	struct espx_net_key key = {};
	struct espx_net_stats *st;
	struct espx_net_stats fresh = {};
	__u32 pid;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	pid = bpf_get_current_pid_tgid() >> 32;
	if (!espx_target_role(&target_pids, pid))
		return 0;

	key.pid = pid;
	key.dport = 0;
	st = bpf_map_lookup_elem(&net_stats, &key);
	if (!st) {
		bpf_map_update_elem(&net_stats, &key, &fresh, BPF_NOEXIST);
		st = bpf_map_lookup_elem(&net_stats, &key);
	}
	if (st)
		st->retrans++;
	return 0;
}

SEC("tracepoint/sched/sched_process_fork")
int espx_sched_process_fork(struct trace_event_raw_sched_process_fork *ctx)
{
	struct espx_config *cfg;
	struct espx_pid_stats *st;
	__u32 tgid;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	tgid = bpf_get_current_pid_tgid() >> 32;
	role = espx_target_role(&target_pids, tgid);
	if (!role)
		return 0;

	st = pid_stats_mut(tgid, role);
	if (st)
		st->thread_fork++;
	return 0;
}

SEC("tracepoint/sched/sched_process_exit")
int espx_sched_process_exit(struct trace_event_raw_sched_process_exit *ctx)
{
	struct espx_config *cfg;
	struct espx_pid_stats *st;
	__u32 tgid;
	__u8 role;

	cfg = probe_config();
	if (!cfg || !cfg->enabled)
		return 0;

	tgid = bpf_get_current_pid_tgid() >> 32;
	role = espx_target_role(&target_pids, tgid);
	if (!role)
		return 0;

	st = pid_stats_mut(tgid, role);
	if (st)
		st->thread_exit++;
	return 0;
}
