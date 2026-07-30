


#ifndef ESPX_PROBE_H
#define ESPX_PROBE_H

#define ESPX_ROLE_TRACKER 1
#define ESPX_ROLE_NGINX 2
#define ESPX_ROLE_REDIS 3
#define ESPX_ROLE_K6 4
#define ESPX_ROLE_PROCESSOR 5

#define ESPX_MARKER_PROCESS_TRACK_ENTER 1
#define ESPX_MARKER_PROCESS_TRACK_EXIT 2
#define ESPX_MARKER_FILTER_CHECK_ENTER 3
#define ESPX_MARKER_FILTER_CHECK_EXIT 4

#define ESPX_SLOW_KIND_SYSCALL 1
#define ESPX_SLOW_KIND_UPROBE 2

#define ESPX_HIST_BUCKETS 32
#define ESPX_SLOW_SYSCALL_NS 10000000ULL  
#define ESPX_DEFAULT_SAMPLE_RATE 1

struct espx_hist {
	__u64 buckets[ESPX_HIST_BUCKETS];
	__u64 count;
	__u64 sum_ns;
	__u64 max_ns;
};

struct espx_pid_stats {
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


#define ESPX_NR_read 0
#define ESPX_NR_write 1
#define ESPX_NR_writev 19
#define ESPX_NR_connect 42
#define ESPX_NR_fsync 74
#define ESPX_NR_fdatasync 75
#define ESPX_NR_sendto 44
#define ESPX_NR_recvfrom 45
#define ESPX_NR_futex 202
#define ESPX_NR_epoll_wait 232

static __always_inline int espx_is_hot_syscall(long syscall_id)
{
	switch (syscall_id) {
	case ESPX_NR_read:
	case ESPX_NR_write:
	case ESPX_NR_writev:
	case ESPX_NR_fsync:
	case ESPX_NR_fdatasync:
	case ESPX_NR_connect:
	case ESPX_NR_sendto:
	case ESPX_NR_recvfrom:
	case ESPX_NR_futex:
	case ESPX_NR_epoll_wait:
		return 1;
	default:
		return 0;
	}
}
#define ESPX_NR_close 3
#define ESPX_NR_dup 32
#define ESPX_NR_socket 41
#define ESPX_NR_accept 43
#define ESPX_NR_openat 257
#define ESPX_NR_accept4 288
#define ESPX_NR_dup3 292
#define ESPX_NR_pipe2 293

static __always_inline void espx_account_fd_exit(struct espx_pid_stats *st, long syscall_id, long ret)
{
	if (!st)
		return;
	if (syscall_id == ESPX_NR_close) {
		if (ret == 0)
			st->fd_close++;
		return;
	}
	if (ret < 0)
		return;
	switch (syscall_id) {
	case ESPX_NR_openat:
	case ESPX_NR_dup:
	case ESPX_NR_dup3:
	case ESPX_NR_pipe2:
		st->fd_open++;
		break;
	case ESPX_NR_socket:
		st->fd_open++;
		st->socket_open++;
		break;
	case ESPX_NR_accept:
	case ESPX_NR_accept4:
		st->fd_open++;
		st->socket_accept++;
		break;
	default:
		break;
	}
}

struct espx_syscall_key {
	__u32 pid;
	__u32 syscall_id;
};

struct espx_syscall_hist_key {
	__u32 pid;
	__u32 syscall_id;
};

struct espx_net_key {
	__u32 pid;
	__u16 dport;
	__u16 _pad;
};

struct espx_net_stats {
	__u64 connects;
	__u64 connect_ns_sum;
	__u64 connect_samples;
	__u64 retrans;
	__u64 rst;
};

struct espx_slow_event {
	__u64 ts_ns;
	__u32 pid;
	__u32 syscall_id;
	__u64 duration_ns;
	__u8 role;
	__u8 kind; 
	__u16 campaign_slot;
	__u32 marker_id;
};

struct espx_config {
	__u32 sample_rate;
	__u32 slow_syscall_ns;
	__u32 enabled;
	__u32 _pad;
};

struct espx_marker_ts_key {
	__u64 pid_tgid;
	__u32 marker_id;
	__u32 _pad;
};

struct espx_marker_hist_key {
	__u32 pid;
	__u32 marker_id;
	__u32 campaign_slot;
	__u32 _pad;
};

static __always_inline int espx_target_role(void *targets, __u32 pid)
{
	__u8 *role;

	if (!targets || !pid)
		return 0;
	role = bpf_map_lookup_elem(targets, &pid);
	if (!role || !*role)
		return 0;
	return (int)*role;
}

static __always_inline int espx_cgroup_role(void *cgroups, __u64 cgid)
{
	__u8 *role;

	if (!cgroups || !cgid)
		return 0;
	role = bpf_map_lookup_elem(cgroups, &cgid);
	if (!role || !*role)
		return 0;
	return (int)*role;
}

static __always_inline int espx_resolve_role(void *targets, void *cgroups, __u32 pid)
{
	__u64 cgid;
	__u8 role;

	cgid = bpf_get_current_cgroup_id();
	if (cgid) {
		role = espx_cgroup_role(cgroups, cgid);
		if (role)
			return role;
	}
	return espx_target_role(targets, pid);
}

static __always_inline int espx_should_sample(const struct espx_config *cfg)
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

static __always_inline void espx_hist_record(struct espx_hist *hist, __u64 delta_ns)
{
	__u32 bucket;
	__u64 v;

	if (!hist || !delta_ns)
		return;
	v = delta_ns;
	bucket = 0;
	while (bucket + 1 < ESPX_HIST_BUCKETS && v > 1) {
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
