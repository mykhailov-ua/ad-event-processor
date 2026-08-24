

#ifndef AD_EVENT_PROCESSOR_PROBE_H
#define AD_EVENT_PROCESSOR_PROBE_H

#include <bpf/bpf_endian.h>

#define AD_EVENT_PROCESSOR_ROLE_TRACKER 1
#define AD_EVENT_PROCESSOR_ROLE_NGINX 2
#define AD_EVENT_PROCESSOR_ROLE_REDIS 3
#define AD_EVENT_PROCESSOR_ROLE_K6 4
#define AD_EVENT_PROCESSOR_ROLE_PROCESSOR 5

#define AD_EVENT_PROCESSOR_MARKER_PROCESS_TRACK_ENTER 1
#define AD_EVENT_PROCESSOR_MARKER_PROCESS_TRACK_EXIT 2
#define AD_EVENT_PROCESSOR_MARKER_FILTER_CHECK_ENTER 3
#define AD_EVENT_PROCESSOR_MARKER_FILTER_CHECK_EXIT 4

#define AD_EVENT_PROCESSOR_SLOW_KIND_SYSCALL 1
#define AD_EVENT_PROCESSOR_SLOW_KIND_UPROBE 2

#define AD_EVENT_PROCESSOR_HIST_BUCKETS 32
#define AD_EVENT_PROCESSOR_SLOW_SYSCALL_NS 10000000ULL  
#define AD_EVENT_PROCESSOR_DEFAULT_SAMPLE_RATE 1

struct probe_hist {
	__u64 buckets[AD_EVENT_PROCESSOR_HIST_BUCKETS];
	__u64 count;
	__u64 sum_ns;
	__u64 max_ns;
};

struct probe_pid_stats {
	__u8 role;
	__u8 _pad[7];
	__u64 ctx_switch_out;
	__u64 ctx_switch_in;
	__u64 voluntary_ctx;
	__u64 involuntary_ctx;
	__u64 runqueue_ns;
	__u64 runqueue_samples;
	__u64 oncpu_ns;
	__u64 last_oncpu_ns;
	__u64 major_faults;
	__u64 fd_open;
	__u64 fd_close;
	__u64 socket_open;
	__u64 socket_accept;
	__u64 thread_fork;
	__u64 thread_exit;
	__u64 minor_faults;
};


#define AD_EVENT_PROCESSOR_NR_read 0
#define AD_EVENT_PROCESSOR_NR_write 1
#define AD_EVENT_PROCESSOR_NR_writev 19
#define AD_EVENT_PROCESSOR_NR_connect 42
#define AD_EVENT_PROCESSOR_NR_fsync 74
#define AD_EVENT_PROCESSOR_NR_fdatasync 75
#define AD_EVENT_PROCESSOR_NR_sendto 44
#define AD_EVENT_PROCESSOR_NR_recvfrom 45
#define AD_EVENT_PROCESSOR_NR_futex 202
#define AD_EVENT_PROCESSOR_NR_epoll_wait 232

#define AD_EVENT_PROCESSOR_AF_INET 2
#define AD_EVENT_PROCESSOR_PG_PORT 5432

static __always_inline __u16 probe_read_sockaddr_port(void *addr)
{
	__u16 family;
	__u16 port_be;

	if (!addr)
		return 0;
	if (bpf_probe_read_user(&family, sizeof(family), addr) < 0)
		return 0;
	if (family != AD_EVENT_PROCESSOR_AF_INET)
		return 0;
	if (bpf_probe_read_user(&port_be, sizeof(port_be), (char *)addr + 2) < 0)
		return 0;
	return bpf_ntohs(port_be);
}

static __always_inline int probe_is_hot_syscall(long syscall_id)
{
	switch (syscall_id) {
	case AD_EVENT_PROCESSOR_NR_read:
	case AD_EVENT_PROCESSOR_NR_write:
	case AD_EVENT_PROCESSOR_NR_writev:
	case AD_EVENT_PROCESSOR_NR_fsync:
	case AD_EVENT_PROCESSOR_NR_fdatasync:
	case AD_EVENT_PROCESSOR_NR_connect:
	case AD_EVENT_PROCESSOR_NR_sendto:
	case AD_EVENT_PROCESSOR_NR_recvfrom:
	case AD_EVENT_PROCESSOR_NR_futex:
	case AD_EVENT_PROCESSOR_NR_epoll_wait:
		return 1;
	default:
		return 0;
	}
}
#define AD_EVENT_PROCESSOR_NR_close 3
#define AD_EVENT_PROCESSOR_NR_dup 32
#define AD_EVENT_PROCESSOR_NR_socket 41
#define AD_EVENT_PROCESSOR_NR_accept 43
#define AD_EVENT_PROCESSOR_NR_openat 257
#define AD_EVENT_PROCESSOR_NR_accept4 288
#define AD_EVENT_PROCESSOR_NR_dup3 292
#define AD_EVENT_PROCESSOR_NR_pipe2 293

static __always_inline void probe_account_fd_exit(struct probe_pid_stats *st, long syscall_id, long ret)
{
	if (!st)
		return;
	if (syscall_id == AD_EVENT_PROCESSOR_NR_close) {
		if (ret == 0)
			st->fd_close++;
		return;
	}
	if (ret < 0)
		return;
	switch (syscall_id) {
	case AD_EVENT_PROCESSOR_NR_openat:
	case AD_EVENT_PROCESSOR_NR_dup:
	case AD_EVENT_PROCESSOR_NR_dup3:
	case AD_EVENT_PROCESSOR_NR_pipe2:
		st->fd_open++;
		break;
	case AD_EVENT_PROCESSOR_NR_socket:
		st->fd_open++;
		st->socket_open++;
		break;
	case AD_EVENT_PROCESSOR_NR_accept:
	case AD_EVENT_PROCESSOR_NR_accept4:
		st->fd_open++;
		st->socket_accept++;
		break;
	default:
		break;
	}
}

struct probe_syscall_key {
	__u32 pid;
	__u32 syscall_id;
};

struct probe_syscall_hist_key {
	__u32 pid;
	__u32 syscall_id;
};

struct probe_net_key {
	__u32 pid;
	__u16 dport;
	__u16 _pad;
};

struct probe_net_stats {
	__u64 connects;
	__u64 connect_ns_sum;
	__u64 connect_samples;
	__u64 retrans;
	__u64 rst;
	__u64 sendto_calls;
	__u64 sendto_bytes;
};

struct probe_syscall_peer {
	__u16 dport;
	__u16 _pad;
	__u32 sendto_len;
};

struct probe_slow_event {
	__u64 ts_ns;
	__u32 pid;
	__u32 syscall_id;
	__u64 duration_ns;
	__u8 role;
	__u8 kind; 
	__u16 campaign_slot;
	__u32 marker_id;
};

struct probe_config {
	__u32 sample_rate;
	__u32 slow_syscall_ns;
	__u32 enabled;
	__u32 _pad;
};

struct probe_marker_ts_key {
	__u64 pid_tgid;
	__u32 marker_id;
	__u32 _pad;
};

struct probe_marker_hist_key {
	__u32 pid;
	__u32 marker_id;
	__u32 campaign_slot;
	__u32 _pad;
};

static __always_inline int probe_target_role(void *targets, __u32 pid)
{
	__u8 *role;

	if (!targets || !pid)
		return 0;
	role = bpf_map_lookup_elem(targets, &pid);
	if (!role || !*role)
		return 0;
	return (int)*role;
}

static __always_inline int probe_cgroup_role(void *cgroups, __u64 cgid)
{
	__u8 *role;

	if (!cgroups || !cgid)
		return 0;
	role = bpf_map_lookup_elem(cgroups, &cgid);
	if (!role || !*role)
		return 0;
	return (int)*role;
}

static __always_inline int probe_resolve_role(void *targets, void *cgroups, __u32 pid)
{
	__u64 cgid;
	__u8 role;

	cgid = bpf_get_current_cgroup_id();
	if (cgid) {
		role = probe_cgroup_role(cgroups, cgid);
		if (role)
			return role;
	}
	return probe_target_role(targets, pid);
}

static __always_inline int probe_should_sample(const struct probe_config *cfg)
{
	__u32 rate;

	if (!cfg || !cfg->enabled)
		return 0;
	rate = cfg->sample_rate;
	if (!rate)
		rate = 1;
	if (rate == 1)
		return 1;
	return (bpf_get_prandom_u32() % rate) == 0;
}

static __always_inline void probe_hist_record(struct probe_hist *hist, __u64 delta_ns)
{
	__u32 bucket;
	__u64 v;

	if (!hist || !delta_ns)
		return;
	v = delta_ns;
	bucket = 0;
	while (bucket + 1 < AD_EVENT_PROCESSOR_HIST_BUCKETS && v > 1) {
		v >>= 1;
		bucket++;
	}
	hist->buckets[bucket]++;
	hist->count++;
	hist->sum_ns += delta_ns;
	if (delta_ns > hist->max_ns)
		hist->max_ns = delta_ns;
}

#endif 
