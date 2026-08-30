/* XDP edge_filter program (deploy/edge/xdp/bpf/edge_filter.c).
 * Attached by cmd/edge-xdp on INGRESS_INTERFACE; maps synced from Redis via edge-bpf-sync.
 * License JWT feature: features.ebpf_xdp_edge (not on single_vps SKU).
 *
 * Tracker-bound traffic: dest TCP port TRACKER_INGRESS_PORT (default 8180).
 *
 * IPv4 lookup order (xdp_edge_filter):
 *   1. allow_v4 LPM (/32 host key)
 *   2. blocklist_host_v4 LRU HASH (/32)
 *   3. blocklist_v4 LPM (/32 host key)
 *   then syn/rst limits, pps token bucket; optional SYN-cookie tail call.
 *
 * IPv6 lookup order (xdp_filter_ipv6_tcp):
 *   1. allow_v6 LPM (/128)
 *   2. blocklist_host_v6 LRU HASH
 *   3. blocklist_v6 LPM (/128)
 *
 * Map max_entries:
 *   blocklist_v4/v6, blocklist_host_v4/v6: 786432
 *   allow_v4/v6: 65536
 *   syn_ratelimit_v4: 786432 per CPU; syn_subnet_ratelimit_v4: 65536 (LRU)
 *   ratelimit_v4, rst_ratelimit_v4: 1048576 per CPU
 *   global_syn: 1 (per-CPU array); config: 1; prog_array: 1
 *   violations, fingerprints ringbufs: 256 KiB each
 *
 * Cross-layer (Redis -> userspace -> kernel maps; parallel L7 path):
 *   control outbox / admin / fraud worker
 *     -> Redis SET blacklist:manual|auto|fraud (+ ZSET blacklist:changelog:add|remove)
 *     -> edge-bpf-sync: LoadPinned maps, BlocklistStore shadow, Update/Delete
 *     -> this program: bpf_map_lookup_elem only (no writes on deny maps)
 *   parallel: edge-blacklist-sync.lua -> ngx.shared _bl_ver / b:{ip} (access-check.lua L7)
 *
 * Memory Model Rules:
 *
 * Allow-before-deny (fail-open for allowlisted host):
 *   IPv4: allow_v4 LPM lookup before any deny map. Match -> XDP_PASS (XDP_STAT_PASS_ALLOWLIST).
 *   Deny never evaluated when allow /32 hits. Userspace skips protected IPs on sync
 *   (edge.IsProtected -> no Update into deny maps).
 *
 * blocklist_host_* LRU HASH (786432 entries):
 *   Host /32 (v4) or full IPv6 addr keys populated by edge-bpf-sync via ebpf.UpdateAny.
 *   Kernel evicts least-recently-used entry when map is full and a new key is inserted.
 *   XDP has no bpf_map_delete_elem on deny maps; no in-program delete path.
 *   Userspace remove path: BlocklistStore.applyHostRemove -> maps.V4Host.Delete (changelog
 *   or full ApplyDiff). LPM deny maps (blocklist_v4/v6) use explicit Delete on remove too.
 *
 * LRU eviction vs userspace shadow (invalidation pattern: bpf_lru_implicit):
 *   When occupied >= max_entries before insert, edge-bpf-sync increments
 *   ad_edge_blocklist_lru_eviction_total (recordLRUEvictionBeforeInsert). Kernel may drop
 *   a cold host entry while BlocklistStore.hosts still lists it until the next 5 min full
 *   SMEMBERS resync (SyncBlocklistFromRedis ApplyDiff) reconciles shadow to Redis truth.
 *
 * L7 generational cache (invalidation pattern: l7_generational_full | l7_generational_incremental):
 *   Not read by this program. ngx.shared bumps _bl_ver on full SMEMBERS sync; incremental
 *   quarantine stamps b:{ip} without version bump. Stale b:{ip} with ip_ver != _bl_ver passes L7.
 *
 * violations ringbuf (256 KiB, BPF_MAP_TYPE_RINGBUF):
 *   emit_violation on SYN / global SYN / SYN-subnet / PPS drops when SYN-cookie tail call
 *   is not taken. Reasons: VIOLATION_SYN=1, VIOLATION_GLOBAL_SYN=2, VIOLATION_PPS=3,
 *   VIOLATION_SYN_SUBNET=4. ringbuf_used_pct + bpf_ringbuf_query before reserve; under pressure
 *   sample VIOLATION_PPS first so SYN/global/subnet alerts retain headroom. Reserve failure
 *   still silent (packet XDP_DROP). edge-bpf-sync drains -> RecordAutoBan -> Redis.
 *
 * fingerprints ringbuf:
 *   emit_fingerprint on SYN when CFG_FLAG_FINGERPRINT set. Aggressive sampling when ring
 *   >60% full; skip when >85% (violations map is separate). Reserve failure silent.
 *   Drain -> edge.Record -> Redis edge:tcp_fp:* (L7 OS fingerprint headers, not blocklist).
 *
 * syn_ratelimit_v4 / ratelimit_v4 (BPF_MAP_TYPE_PERCPU_HASH):
 *   Per-CPU shards avoid LRU spinlock contention under spoofed-IP floods. Limit is enforced
 *   per RSS CPU queue (same as global_syn). syn_subnet_ratelimit_v4 stays LRU (/24 aggregate).
 *
 * prog_array tail call (index PROG_IDX_SYN_COOKIE=0):
 *   try_syn_cookie -> bpf_tail_call(ctx, &prog_array, 0) -> xdp_syn_cookie program.
 *   emit_ipv4_synack: SYN-ACK uses 20-byte TCP (doff=5), window SYNACK_TCP_WINDOW (non-zero);
 *   checksum and ip tot_len use out_tcph_len;
 *   bpf_xdp_adjust_tail trims from ctx->data_end to drop ingress options/payload.
 *   gen_syncookie_ipv4 still reads full ingress tcph_len (options included).
 *   Wired by cmd/edge-xdp wireProgArray when XdpSynCookie object loaded and
 *   CFG_FLAG_SYN_COOKIE set. Tail-call miss (prog not loaded) falls through to XDP_DROP.
 *
 * Verify: make gen bpf-dev; go test ./internal/edge/... -count=1
 */
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/in.h>
#include <linux/in6.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/tcp.h>
#include <linux/udp.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

#ifndef BPF_FUNC_tcp_gen_syncookie_ipv4
#define BPF_FUNC_tcp_gen_syncookie_ipv4 163
#endif

static long (*const bpf_tcp_gen_syncookie_ipv4)(void *iph, __u16 iph_len, void *tcph,
						__u16 tcph_len, __u64 *cookie) =
	(void *)(long)BPF_FUNC_tcp_gen_syncookie_ipv4;

#ifndef TRACKER_INGRESS_PORT
#define TRACKER_INGRESS_PORT 8180
#endif

#define SYN_WINDOW_NS 1000000000ULL
#define NS_PER_SEC 1000000000ULL

#define DEFAULT_SYN_LIMIT 64
#define DEFAULT_PPS_RATE 2000
#define DEFAULT_GLOBAL_SYN_LIMIT 50000
#define DEFAULT_ASSUMED_CPUS 8
#define DEFAULT_SYN_SUBNET_LIMIT 4096
#define DEFAULT_RST_RATE 512
#define DEFAULT_RST_BURST 512
#define SYNACK_TCP_WINDOW 64240

#define PROG_IDX_SYN_COOKIE 0

#define CFG_FLAG_FINGERPRINT 0x01
#define CFG_FLAG_SYN_COOKIE  0x02

#define TCP_FLAG_FIN 0x01
#define TCP_FLAG_SYN 0x02
#define TCP_FLAG_RST 0x04
#define TCP_FLAG_PSH 0x08
#define TCP_FLAG_ACK 0x10
#define TCP_FLAG_URG 0x20

#define VIOLATION_SYN 1
#define VIOLATION_GLOBAL_SYN 2
#define VIOLATION_PPS 3
#define VIOLATION_SYN_SUBNET 4

#define RINGBUF_BYTES (256 * 1024)
#define RINGBUF_FP_SAMPLE_PCT 60
#define RINGBUF_FP_SKIP_PCT 85
#define RINGBUF_VIOLATION_PPS_SAMPLE_PCT 80
#define RINGBUF_VIOLATION_PPS_STOP_PCT 95
#define RINGBUF_VIOLATION_OTHER_SAMPLE_PCT 90

struct fingerprint_event {
	__u64 ts_ns;
	__u32 src_ip;
	__u32 tcp_hash;
	__u16 window;
	__u8 ttl;
	__u8 mss;
};

struct ipv4_lpm_key {
	__u32 prefixlen;
	__u32 addr;
};

struct ipv6_lpm_key {
	__u32 prefixlen;
	__u8 addr[16];
};

struct syn_state {
	__u64 window_start_ns;
	__u32 count;
};

struct pps_bucket {
	__u64 last_ns;
	__u32 tokens;
};

struct sctphdr {
	__be16 source;
	__be16 dest;
};

struct edge_config {
	__u32 syn_limit;
	__u32 pps_rate;
	__u32 global_syn_limit;
	__u32 assumed_cpus;
	__u32 syn_subnet_limit;
	__u32 syn_cookie_enabled;
	__u32 fingerprint_enabled;
};

struct violation_event {
	__u64 ts_ns;
	__u32 src_ip;
	__u8 reason;
	__u8 _pad[3];
};

enum xdp_stats {
	XDP_STAT_PASS = 0,
	XDP_STAT_PASS_ALLOWLIST,
	XDP_STAT_DROP_BLOCKLIST,
	XDP_STAT_DROP_SYN,
	XDP_STAT_DROP_GLOBAL_SYN,
	XDP_STAT_DROP_PPS,
	XDP_STAT_DROP_ANOMALY,
	XDP_STAT_DROP_INVALID,
	XDP_STAT_DROP_NON_TCP,
	XDP_STAT_DROP_RST,
	XDP_STAT_DROP_SYN_SUBNET,
	XDP_STAT_SYN_COOKIE,
	XDP_STAT_FINGERPRINT,
	XDP_STAT_MAX,
};

#define STAT_NONE XDP_STAT_MAX

struct {
	__uint(type, BPF_MAP_TYPE_LPM_TRIE);
	__uint(max_entries, 786432);
	__type(key, struct ipv4_lpm_key);
	__type(value, __u8);
	__uint(map_flags, BPF_F_NO_PREALLOC);
} blocklist_v4 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LPM_TRIE);
	__uint(max_entries, 65536);
	__type(key, struct ipv4_lpm_key);
	__type(value, __u8);
	__uint(map_flags, BPF_F_NO_PREALLOC);
} allow_v4 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LPM_TRIE);
	__uint(max_entries, 786432);
	__type(key, struct ipv6_lpm_key);
	__type(value, __u8);
	__uint(map_flags, BPF_F_NO_PREALLOC);
} blocklist_v6 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 786432);
	__type(key, __u32);
	__type(value, __u8);
} blocklist_host_v4 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 786432);
	__type(key, struct in6_addr);
	__type(value, __u8);
} blocklist_host_v6 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LPM_TRIE);
	__uint(max_entries, 65536);
	__type(key, struct ipv6_lpm_key);
	__type(value, __u8);
	__uint(map_flags, BPF_F_NO_PREALLOC);
} allow_v6 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 786432);
	__type(key, __u32);
	__type(value, struct syn_state);
} syn_ratelimit_v4 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 65536);
	__type(key, __u32);
	__type(value, struct syn_state);
} syn_subnet_ratelimit_v4 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 1048576);
	__type(key, __u32);
	__type(value, struct pps_bucket);
} ratelimit_v4 SEC(".maps");

/* rst_ratelimit_v4: PERCPU_HASH (not LRU) so RST floods do not evict unrelated /32 keys.
 * DEFAULT_RST_RATE/BURST 512/s per CPU shard; token bucket like ratelimit_v4. */
struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_HASH);
	__uint(max_entries, 1048576);
	__type(key, __u32);
	__type(value, struct pps_bucket);
} rst_ratelimit_v4 SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct syn_state);
} global_syn SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, XDP_STAT_MAX);
	__type(key, __u32);
	__type(value, __u64);
} stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct edge_config);
} config SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} violations SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);
} fingerprints SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PROG_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, __u32);
} prog_array SEC(".maps");

static __always_inline void stat_inc(__u32 idx)
{
	__u64 *val = bpf_map_lookup_elem(&stats, &idx);
	if (val)
		(*val)++;
}

static __always_inline void stat_inc_if(__u32 idx)
{
	if (idx >= XDP_STAT_MAX)
		return;
	stat_inc(idx);
}

static __always_inline __u32 ringbuf_used_pct(void *ring)
{
	__u64 size = bpf_ringbuf_query(ring, BPF_RB_RING_SIZE);
	__u64 used = bpf_ringbuf_query(ring, BPF_RB_AVAIL_DATA);

	if (!size)
		return 100;
	if (used >= size)
		return 100;
	return (__u32)((used * 100) / size);
}

static __always_inline int ringbuf_has_record_space(void *ring, __u32 rec_sz)
{
	__u64 size = bpf_ringbuf_query(ring, BPF_RB_RING_SIZE);
	__u64 used = bpf_ringbuf_query(ring, BPF_RB_AVAIL_DATA);

	if (!size)
		return 0;
	if (used + rec_sz + BPF_RINGBUF_HDR_SZ > size)
		return 0;
	return 1;
}

static __always_inline int ringbuf_reserve_allowed(void *ring, __u32 used_pct, __u8 priority)
{
	if (!ringbuf_has_record_space(ring, sizeof(struct violation_event)))
		return 0;
	if (priority == 1 && used_pct >= RINGBUF_VIOLATION_PPS_STOP_PCT)
		return 0;
	return 1;
}

static __always_inline int ringbuf_reserve_allowed_fp(void *ring, __u32 used_pct)
{
	if (!ringbuf_has_record_space(ring, sizeof(struct fingerprint_event)))
		return 0;
	if (used_pct >= RINGBUF_FP_SKIP_PCT)
		return 0;
	return 1;
}

static __always_inline void emit_violation(__u32 src_ip, __u8 reason)
{
	struct violation_event *evt;
	__u32 used_pct = ringbuf_used_pct(&violations);
	__u8 priority = 2;

	if (reason == VIOLATION_PPS)
		priority = 1;
	if (used_pct >= RINGBUF_VIOLATION_PPS_SAMPLE_PCT && reason == VIOLATION_PPS) {
		if ((src_ip ^ reason) & 0x7)
			return;
	}
	if (used_pct >= RINGBUF_VIOLATION_OTHER_SAMPLE_PCT && reason != VIOLATION_GLOBAL_SYN) {
		if ((src_ip ^ reason) & 0x3)
			return;
	}
	if (!ringbuf_reserve_allowed(&violations, used_pct, priority))
		return;

	evt = bpf_ringbuf_reserve(&violations, sizeof(*evt), 0);
	if (!evt)
		return;
	evt->ts_ns = bpf_ktime_get_ns();
	evt->src_ip = src_ip;
	evt->reason = reason;
	bpf_ringbuf_submit(evt, 0);
}

static __always_inline __u8 read_tcp_mss(struct tcphdr *tcph, void *data_end)
{
	__u32 doff = tcph->doff * 4;

	if (doff <= sizeof(*tcph))
		return 0;
	if ((__u8 *)tcph + doff > (__u8 *)data_end)
		return 0;

	__u8 *p = (__u8 *)tcph + sizeof(*tcph);
	if (p + 4 > (__u8 *)data_end)
		return 0;
	if (p[0] != 2 || p[1] < 4)
		return 0;
	return (__u8)(bpf_ntohs(*(__be16 *)(p + 2)) >> 8);
}

static __always_inline __u32 hash_tcp_syn_fields(__u8 ttl, __u16 window, __u8 mss, __u8 doff)
{
	__u32 h = ttl;

	h = (h << 5) ^ window;
	h = (h << 5) ^ mss;
	h = (h << 3) ^ doff;
	return h;
}

static __always_inline void emit_fingerprint(__u64 now, __u32 src_ip, __u32 tcp_hash,
					     __u16 window, __u8 ttl, __u8 mss)
{
	struct fingerprint_event *evt;
	__u32 used_pct = ringbuf_used_pct(&fingerprints);

	if (used_pct >= RINGBUF_FP_SAMPLE_PCT) {
		if ((src_ip ^ (__u32)(now >> 20)) & 0xF)
			return;
	}
	if (!ringbuf_reserve_allowed_fp(&fingerprints, used_pct))
		return;

	evt = bpf_ringbuf_reserve(&fingerprints, sizeof(*evt), 0);
	if (!evt)
		return;
	evt->ts_ns = now;
	evt->src_ip = src_ip;
	evt->window = window;
	evt->ttl = ttl;
	evt->mss = mss;
	evt->tcp_hash = tcp_hash;
	bpf_ringbuf_submit(evt, 0);
}

static __always_inline __u32 ipv4_subnet24(__u32 addr)
{
	return addr & bpf_htonl(0xFFFFFF00);
}


static __always_inline __u8 load_config_scalars(__u32 *syn_limit, __u32 *pps_rate,
						__u32 *global_syn_limit, __u32 *assumed_cpus,
						__u32 *syn_subnet_limit)
{
	__u32 key = 0;
	struct edge_config *map_cfg = bpf_map_lookup_elem(&config, &key);
	__u8 flags = CFG_FLAG_FINGERPRINT;

	*syn_limit = DEFAULT_SYN_LIMIT;
	*pps_rate = DEFAULT_PPS_RATE;
	*global_syn_limit = DEFAULT_GLOBAL_SYN_LIMIT;
	*assumed_cpus = DEFAULT_ASSUMED_CPUS;
	*syn_subnet_limit = DEFAULT_SYN_SUBNET_LIMIT;

	if (!map_cfg || map_cfg->assumed_cpus == 0)
		return flags;

	if (map_cfg->syn_limit)
		*syn_limit = map_cfg->syn_limit;
	if (map_cfg->pps_rate)
		*pps_rate = map_cfg->pps_rate;
	if (map_cfg->global_syn_limit)
		*global_syn_limit = map_cfg->global_syn_limit;
	*assumed_cpus = map_cfg->assumed_cpus;
	if (map_cfg->syn_subnet_limit)
		*syn_subnet_limit = map_cfg->syn_subnet_limit;
	if (map_cfg->syn_cookie_enabled)
		flags |= CFG_FLAG_SYN_COOKIE;
	if (!map_cfg->fingerprint_enabled)
		flags &= ~CFG_FLAG_FINGERPRINT;
	return flags;
}

static __always_inline void swap_eth_addrs(__u8 *a, __u8 *b)
{
	__u8 tmp[6];

	__builtin_memcpy(tmp, a, 6);
	__builtin_memcpy(a, b, 6);
	__builtin_memcpy(b, tmp, 6);
}

static __always_inline __u16 csum_fold(__u32 csum)
{
	csum = (csum & 0xffff) + (csum >> 16);
	csum = (csum & 0xffff) + (csum >> 16);
	return (__u16)~csum;
}

static __always_inline __u16 csum_tcpudp_magic(__be32 saddr, __be32 daddr,
					       __u32 len, __u8 proto,
					       __u32 csum)
{
	__u64 s = csum;

	s += (__u32)saddr;
	s += (__u32)daddr;
#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__
	s += (proto + len) << 8;
#else
	s += proto + len;
#endif
	s = (s & 0xffffffff) + (s >> 32);
	s = (s & 0xffffffff) + (s >> 32);
	return (__u16)s;
}

static __always_inline __u32 gen_syncookie_ipv4(struct iphdr *iph, __u32 ihl_len,
						struct tcphdr *tcph, __u32 tcph_len,
						__u64 *cookie_out)
{
	__s64 raw;

	raw = bpf_tcp_raw_gen_syncookie_ipv4(iph, tcph, tcph_len);
	if (raw >= 0)
		return (__u32)raw;

	if (bpf_tcp_gen_syncookie_ipv4(iph, ihl_len, tcph, tcph_len, cookie_out) >= 0)
		return (__u32)*cookie_out;

	return 0;
}

static __always_inline int emit_ipv4_synack(struct xdp_md *ctx,
					    struct ethhdr *eth,
					    struct iphdr *iph,
					    struct tcphdr *tcph,
					    __u32 ihl_len, __u32 tcph_len,
					    __u32 cookie)
{
	void *data_end = (void *)(long)ctx->data_end;
	__s64 csum_val;
	__u32 old_len;
	__u32 new_len;
	__u32 out_tcph_len = sizeof(*tcph);
	__be32 tmp_ip;
	__be16 tmp_port;

	swap_eth_addrs(eth->h_source, eth->h_dest);

	tmp_ip = iph->saddr;
	iph->saddr = iph->daddr;
	iph->daddr = tmp_ip;
	iph->tot_len = bpf_htons(ihl_len + out_tcph_len);
	iph->check = 0;

	tmp_port = tcph->source;
	tcph->source = tcph->dest;
	tcph->dest = tmp_port;
	tcph->ack_seq = bpf_htonl(bpf_ntohl(tcph->seq) + 1);
	tcph->seq = bpf_htonl(cookie);
	*(__u8 *)((__u8 *)tcph + 13) = TCP_FLAG_SYN | TCP_FLAG_ACK;
	tcph->doff = 5;
	tcph->window = bpf_htons(SYNACK_TCP_WINDOW);
	tcph->urg_ptr = 0;
	tcph->check = 0;

	csum_val = bpf_csum_diff(NULL, 0, (__be32 *)tcph, out_tcph_len, 0);
	if (csum_val < 0)
		return XDP_DROP;
	tcph->check = csum_tcpudp_magic(iph->saddr, iph->daddr, out_tcph_len,
					IPPROTO_TCP, (__u32)csum_val);

	csum_val = bpf_csum_diff(NULL, 0, (__be32 *)iph, ihl_len, 0);
	if (csum_val < 0)
		return XDP_DROP;
	iph->check = csum_fold((__u32)csum_val);

	old_len = (__u8 *)data_end - (__u8 *)eth;
	new_len = sizeof(*eth) + ihl_len + out_tcph_len;
	if (new_len != old_len) {
		if (bpf_xdp_adjust_tail(ctx, (__s32)new_len - (__s32)old_len))
			return XDP_DROP;
	}

	return XDP_TX;
}

static __always_inline int try_syn_cookie(struct xdp_md *ctx)
{
	__u32 idx = PROG_IDX_SYN_COOKIE;

	bpf_tail_call(ctx, &prog_array, idx);
	return XDP_DROP;
}

static __always_inline int check_syn_limit(__u32 src_ip, __u64 now, __u32 syn_limit)
{
	struct syn_state *st = bpf_map_lookup_elem(&syn_ratelimit_v4, &src_ip);
	struct syn_state new_st = {};

	if (st) {
		if (now - st->window_start_ns < SYN_WINDOW_NS) {
			if (st->count >= syn_limit)
				return XDP_DROP;
			new_st.window_start_ns = st->window_start_ns;
			new_st.count = st->count + 1;
		} else {
			new_st.window_start_ns = now;
			new_st.count = 1;
		}
	} else {
		new_st.window_start_ns = now;
		new_st.count = 1;
	}

	bpf_map_update_elem(&syn_ratelimit_v4, &src_ip, &new_st, BPF_ANY);
	return XDP_PASS;
}

/* check_syn_subnet_limit: keys are IPv4 /24 (CGNAT); DEFAULT_SYN_SUBNET_LIMIT 4096. */
static __always_inline int check_syn_subnet_limit(__u32 src_ip, __u64 now, __u32 subnet_limit)
{
	__u32 subnet = ipv4_subnet24(src_ip);
	struct syn_state *st = bpf_map_lookup_elem(&syn_subnet_ratelimit_v4, &subnet);
	struct syn_state new_st = {};

	if (st) {
		if (now - st->window_start_ns < SYN_WINDOW_NS) {
			if (st->count >= subnet_limit)
				return XDP_DROP;
			new_st.window_start_ns = st->window_start_ns;
			new_st.count = st->count + 1;
		} else {
			new_st.window_start_ns = now;
			new_st.count = 1;
		}
	} else {
		new_st.window_start_ns = now;
		new_st.count = 1;
	}

	bpf_map_update_elem(&syn_subnet_ratelimit_v4, &subnet, &new_st, BPF_ANY);
	return XDP_PASS;
}

static __always_inline int check_global_syn(__u64 now, __u32 global_limit, __u32 assumed_cpus)
{
	__u32 key = 0;
	__u32 per_cpu = global_limit / assumed_cpus;
	struct syn_state *st = bpf_map_lookup_elem(&global_syn, &key);
	struct syn_state new_st = {};

	if (!st || per_cpu == 0)
		return XDP_PASS;

	if (now - st->window_start_ns < SYN_WINDOW_NS) {
		if (st->count >= per_cpu)
			return XDP_DROP;
		new_st.window_start_ns = st->window_start_ns;
		new_st.count = st->count + 1;
	} else {
		new_st.window_start_ns = now;
		new_st.count = 1;
	}

	bpf_map_update_elem(&global_syn, &key, &new_st, BPF_ANY);
	return XDP_PASS;
}

static __always_inline int check_pps_limit(__u32 src_ip, __u64 now, __u32 pps_rate)
{
	struct pps_bucket *st = bpf_map_lookup_elem(&ratelimit_v4, &src_ip);
	struct pps_bucket new_st = {};
	__u32 burst = pps_rate;
	__u32 tokens = burst;

	if (st) {
		tokens = st->tokens;
		__u64 elapsed = now - st->last_ns;
		if (elapsed > NS_PER_SEC)
			elapsed = NS_PER_SEC;
		if (elapsed > 0) {
			__u64 added = (elapsed * pps_rate) / NS_PER_SEC;
			if (added > 0) {
				tokens += (__u32)added;
				if (tokens > burst)
					tokens = burst;
			}
		}
	}

	if (!tokens)
		return XDP_DROP;

	new_st.last_ns = now;
	new_st.tokens = tokens - 1;
	bpf_map_update_elem(&ratelimit_v4, &src_ip, &new_st, BPF_ANY);
	return XDP_PASS;
}

/* check_rst_limit: inbound RST toward tracker port; PERCPU rst_ratelimit_v4 shard. */
static __always_inline int check_rst_limit(__u32 src_ip, __u64 now)
{
	struct pps_bucket *st = bpf_map_lookup_elem(&rst_ratelimit_v4, &src_ip);
	struct pps_bucket new_st = {};
	__u32 tokens = DEFAULT_RST_BURST;

	if (st) {
		tokens = st->tokens;
		__u64 elapsed = now - st->last_ns;
		if (elapsed > NS_PER_SEC)
			elapsed = NS_PER_SEC;
		if (elapsed > 0) {
			__u64 added = (elapsed * DEFAULT_RST_RATE) / NS_PER_SEC;
			if (added > 0) {
				tokens += (__u32)added;
				if (tokens > DEFAULT_RST_BURST)
					tokens = DEFAULT_RST_BURST;
			}
		}
	}

	if (!tokens)
		return XDP_DROP;

	new_st.last_ns = now;
	new_st.tokens = tokens - 1;
	bpf_map_update_elem(&rst_ratelimit_v4, &src_ip, &new_st, BPF_ANY);
	return XDP_PASS;
}

static __always_inline int is_tcp_anomaly_flags(__u8 fl)
{
	if ((fl & (TCP_FLAG_SYN | TCP_FLAG_FIN)) == (TCP_FLAG_SYN | TCP_FLAG_FIN))
		return 1;
	if ((fl & (TCP_FLAG_SYN | TCP_FLAG_RST)) == (TCP_FLAG_SYN | TCP_FLAG_RST))
		return 1;
	if (fl == 0)
		return 1;
	if (fl == TCP_FLAG_FIN)
		return 1;
	if ((fl & (TCP_FLAG_FIN | TCP_FLAG_PSH | TCP_FLAG_URG)) ==
	    (TCP_FLAG_FIN | TCP_FLAG_PSH | TCP_FLAG_URG))
		return 1;
	return 0;
}

static __always_inline int is_invalid_tcp(struct tcphdr *tcph)
{
	if (tcph->doff < 5)
		return 1;
	if (tcph->source == 0)
		return 1;
	return 0;
}

static __always_inline int drop_non_tcp_tracker(__u8 proto, void *l4, void *data_end)
{
	if (proto == IPPROTO_UDP) {
		struct udphdr *udph = l4;
		if ((void *)(udph + 1) > data_end)
			return XDP_PASS;
		if (bpf_ntohs(udph->dest) == TRACKER_INGRESS_PORT)
			return XDP_DROP;
		return XDP_PASS;
	}

	if (proto == IPPROTO_SCTP) {
		struct sctphdr *sctph = l4;
		if ((void *)(sctph + 1) > data_end)
			return XDP_PASS;
		if (bpf_ntohs(sctph->dest) == TRACKER_INGRESS_PORT)
			return XDP_DROP;
		return XDP_PASS;
	}

	if (proto == IPPROTO_ICMP)
		return XDP_DROP;

	return XDP_PASS;
}

SEC("xdp")
int xdp_syn_cookie(struct xdp_md *ctx)
{
	void *data = (void *)(long)ctx->data;
	void *data_end = (void *)(long)ctx->data_end;
	struct ethhdr *eth = data;
	struct iphdr *iph;
	struct tcphdr *tcph;
	__u32 ihl_len;
	__u32 tcph_len;
	__u64 cookie_out = 0;
	__u32 cookie;
	int action;

	if ((void *)(eth + 1) > data_end)
		return XDP_DROP;

	iph = (void *)(eth + 1);
	if ((void *)(iph + 1) > data_end)
		return XDP_DROP;

	ihl_len = iph->ihl * 4;
	if (ihl_len < sizeof(*iph))
		return XDP_DROP;

	tcph = (void *)iph + ihl_len;
	if ((void *)(tcph + 1) > data_end)
		return XDP_DROP;

	tcph_len = tcph->doff * 4;
	if (tcph_len < sizeof(*tcph))
		return XDP_DROP;
	if ((__u8 *)tcph + tcph_len > (__u8 *)data_end)
		return XDP_DROP;

	cookie = gen_syncookie_ipv4(iph, ihl_len, tcph, tcph_len, &cookie_out);
	if (!cookie)
		return XDP_DROP;

	action = emit_ipv4_synack(ctx, eth, iph, tcph, ihl_len, tcph_len, cookie);
	if (action != XDP_TX)
		return XDP_DROP;

	stat_inc(XDP_STAT_SYN_COOKIE);
	return XDP_TX;
}

static __always_inline void ipv6_lpm_key_from_addr(struct ipv6_lpm_key *key, const struct in6_addr *addr)
{
	key->prefixlen = 128;
	__builtin_memcpy(key->addr, addr, 16);
}

static __always_inline int xdp_filter_ipv6_tcp(struct ipv6hdr *ip6, struct tcphdr *tcph)
{
	struct ipv6_lpm_key al_key = {};
	ipv6_lpm_key_from_addr(&al_key, &ip6->saddr);
	if (bpf_map_lookup_elem(&allow_v6, &al_key)) {
		stat_inc(XDP_STAT_PASS_ALLOWLIST);
		return XDP_PASS;
	}

	if (bpf_map_lookup_elem(&blocklist_host_v6, &ip6->saddr)) {
		stat_inc(XDP_STAT_DROP_BLOCKLIST);
		return XDP_DROP;
	}

	struct ipv6_lpm_key bl_key = {};
	ipv6_lpm_key_from_addr(&bl_key, &ip6->saddr);
	if (bpf_map_lookup_elem(&blocklist_v6, &bl_key)) {
		stat_inc(XDP_STAT_DROP_BLOCKLIST);
		return XDP_DROP;
	}

	stat_inc(XDP_STAT_PASS);
	return XDP_PASS;
}

SEC("xdp")
int xdp_edge_filter(struct xdp_md *ctx)
{
	void *data = (void *)(long)ctx->data;
	void *data_end = (void *)(long)ctx->data_end;
	__u32 action = XDP_PASS;
	__u32 stat_idx = STAT_NONE;
	__u8 fp_stat = 0;

	struct ethhdr *eth = data;
	if ((void *)(eth + 1) > data_end)
		return XDP_PASS;

	if (eth->h_proto != bpf_htons(ETH_P_IP)) {
		if (eth->h_proto != bpf_htons(ETH_P_IPV6))
			return XDP_PASS;

		struct ipv6hdr *ip6 = (void *)(eth + 1);
		if ((void *)(ip6 + 1) > data_end)
			return XDP_PASS;
		if (ip6->nexthdr != IPPROTO_TCP)
			return XDP_PASS;

		struct tcphdr *tcph = (void *)(ip6 + 1);
		if ((void *)(tcph + 1) > data_end)
			return XDP_PASS;
		if (bpf_ntohs(tcph->dest) != TRACKER_INGRESS_PORT)
			return XDP_PASS;

		return xdp_filter_ipv6_tcp(ip6, tcph);
	}

	struct iphdr *iph = (void *)(eth + 1);
	if ((void *)(iph + 1) > data_end)
		return XDP_PASS;

	__u32 ihl_len = iph->ihl * 4;
	if (ihl_len < sizeof(*iph))
		return XDP_PASS;

	void *l4 = (void *)iph + ihl_len;

	if (iph->protocol != IPPROTO_TCP) {
		action = drop_non_tcp_tracker(iph->protocol, l4, data_end);
		if (action == XDP_DROP) {
			stat_idx = XDP_STAT_DROP_NON_TCP;
			goto out;
		}
		return XDP_PASS;
	}

	struct tcphdr *tcph = l4;
	if ((void *)(tcph + 1) > data_end)
		return XDP_PASS;

	if (bpf_ntohs(tcph->dest) != TRACKER_INGRESS_PORT)
		return XDP_PASS;

	__u32 src_ip = iph->saddr;

	struct ipv4_lpm_key al_key = {
		.prefixlen = 32,
		.addr = src_ip,
	};
	if (bpf_map_lookup_elem(&allow_v4, &al_key)) {
		stat_idx = XDP_STAT_PASS_ALLOWLIST;
		goto out;
	}

	if (bpf_map_lookup_elem(&blocklist_host_v4, &src_ip)) {
		action = XDP_DROP;
		stat_idx = XDP_STAT_DROP_BLOCKLIST;
		goto out;
	}

	struct ipv4_lpm_key bl_key = {
		.prefixlen = 32,
		.addr = src_ip,
	};
	if (bpf_map_lookup_elem(&blocklist_v4, &bl_key)) {
		action = XDP_DROP;
		stat_idx = XDP_STAT_DROP_BLOCKLIST;
		goto out;
	}

	__u8 tcp_fl = *(__u8 *)((__u8 *)tcph + 13);

	if (is_tcp_anomaly_flags(tcp_fl)) {
		action = XDP_DROP;
		stat_idx = XDP_STAT_DROP_ANOMALY;
		goto out;
	}

	if (is_invalid_tcp(tcph)) {
		action = XDP_DROP;
		stat_idx = XDP_STAT_DROP_INVALID;
		goto out;
	}

	__u32 syn_limit, pps_rate, global_syn_limit, assumed_cpus, syn_subnet_limit;
	__u8 cfg_flags = load_config_scalars(&syn_limit, &pps_rate, &global_syn_limit,
					     &assumed_cpus, &syn_subnet_limit);

	__u64 now = bpf_ktime_get_ns();

	if (tcp_fl & TCP_FLAG_RST) {
		if (check_rst_limit(src_ip, now) == XDP_DROP) {
			action = XDP_DROP;
			stat_idx = XDP_STAT_DROP_RST;
			goto out;
		}
	}

	if ((tcp_fl & (TCP_FLAG_SYN | TCP_FLAG_ACK)) == TCP_FLAG_SYN) {
		if (cfg_flags & CFG_FLAG_FINGERPRINT) {
			__u16 win = bpf_ntohs(tcph->window);
			__u8 mss = 0;

			if (tcph->doff > 5)
				mss = read_tcp_mss(tcph, data_end);
			__u32 hash = hash_tcp_syn_fields(iph->ttl, win, mss, tcph->doff);

			emit_fingerprint(now, src_ip, hash, win, iph->ttl, mss);
			fp_stat = 1;
		}
		if (check_global_syn(now, global_syn_limit, assumed_cpus) == XDP_DROP) {
			if (cfg_flags & CFG_FLAG_SYN_COOKIE)
				return try_syn_cookie(ctx);
			emit_violation(src_ip, VIOLATION_GLOBAL_SYN);
			action = XDP_DROP;
			stat_idx = XDP_STAT_DROP_GLOBAL_SYN;
			goto out;
		}
		if (check_syn_subnet_limit(src_ip, now, syn_subnet_limit) == XDP_DROP) {
			if (cfg_flags & CFG_FLAG_SYN_COOKIE)
				return try_syn_cookie(ctx);
			emit_violation(src_ip, VIOLATION_SYN_SUBNET);
			action = XDP_DROP;
			stat_idx = XDP_STAT_DROP_SYN_SUBNET;
			goto out;
		}
		if (check_syn_limit(src_ip, now, syn_limit) == XDP_DROP) {
			if (cfg_flags & CFG_FLAG_SYN_COOKIE)
				return try_syn_cookie(ctx);
			emit_violation(src_ip, VIOLATION_SYN);
			action = XDP_DROP;
			stat_idx = XDP_STAT_DROP_SYN;
			goto out;
		}
	}

	if (check_pps_limit(src_ip, now, pps_rate) == XDP_DROP) {
		emit_violation(src_ip, VIOLATION_PPS);
		action = XDP_DROP;
		stat_idx = XDP_STAT_DROP_PPS;
		goto out;
	}

	stat_idx = XDP_STAT_PASS;

out:
	stat_inc_if(stat_idx);
	if (fp_stat)
		stat_inc(XDP_STAT_FINGERPRINT);
	return action;
}

char LICENSE[] SEC("license") = "GPL";
