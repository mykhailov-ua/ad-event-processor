// Package main attaches edge XDP programs and pins BPF maps for edge-bpf-sync.
//
// Role:
//   - Load bpf2go EdgeObjects; pin blocklist/ratelimit/fingerprint maps to BPFPinDir.
//   - Attach xdp_edge_filter when ebpf_xdp_edge license entitles (Redis check).
//   - Wire prog_array slot 0 to xdp_syn_cookie when SYN cookie program is present.
//
// Topology:
//   - Runs on edge host with CAP_NET_ADMIN + CAP_BPF; requires BTF-capable kernel.
//   - edge-bpf-sync (separate binary) populates pinned maps from Redis shard 0.
//   - Nginx Lua remains L7 fallback when XDP attach skipped (unlicensed or attach failure).
//
// Init order (main):
//  1. Parse -iface / INGRESS_INTERFACE, -pin-dir (default /sys/fs/bpf/ad-event-processor), -mode / XDP_MODE.
//  2. rlimit.RemoveMemlock; load BPF objects; InitConfigFromEnv; wireProgArray.
//  3. pinMaps (16 maps); 30s pinned-map count metrics ticker.
//  4. attachXDPWithFallback when licensed; lifecycle.WaitSignal shutdown.
//
// Map pin lifecycle (attach.go pinMaps):
//   - pinDir created 0755; each map: os.Remove(path) then m.Pin(path) under BPFPinDir.
//   - Pinned names: blocklist_v4/v6, blocklist_host_v4/v6, allow_v4/v6, syn_ratelimit_v4,
//     syn_subnet_ratelimit_v4, ratelimit_v4, rst_ratelimit_v4, global_syn, stats, config,
//     violations, fingerprints, prog_array.
//   - edge-bpf-sync opens subset via LoadPinned* (deny/allow/stats/ringbufs); does not attach XDP.
//   - Shutdown: process exit closes link and embedded map fds; pinned paths remain until unlinked
//     (ops: stop bpf-sync, stop edge-xdp, rm pin dir for clean reload).
//
// prog_array tail call:
//   - BPF_MAP_TYPE_PROG_ARRAY max_entries=1; key 0 -> XdpSynCookie program FD (wireProgArray).
//   - edge_filter.c try_syn_cookie: bpf_tail_call(ctx, &prog_array, PROG_IDX_SYN_COOKIE).
//   - Taken when SYN limits exceeded and CFG_FLAG_SYN_COOKIE set (InitConfigFromEnv / SYN cookie env).
//   - Miss (nil XdpSynCookie or update skipped) returns XDP_DROP from try_syn_cookie fall-through.
//
// License gate (idle vs attach):
//   - ebpfEdgeAttachAllowed: edge.EbpfEdgeLicensed via Redis (1s timeout); REDIS_ADDRS unset -> attach
//     allowed with warn (dev only).
//   - Unlicensed: skip attachXDPWithFallback; maps still pinned so edge-bpf-sync can open deny maps
//     but sync loops idle when unlicensed (same EbpfEdgeLicensed check in bpf-sync).
//   - Attach failure when licensed: os.Exit 1 (fail-closed for XDP path; L7 nginx still serves).
//
// Memory Model Rules (coordination with bpf-sync / L7):
//   - Kernel deny maps are read-only in XDP; all invalidation is userspace Update/Delete via bpf-sync.
//   - Allow-before-deny lookup order fixed in edge_filter.c; allowlist synced separately each tick.
//   - L7 ngx.shared mirrors (blacklist_cache, edge_config, slot_map) are independent of BPF maps;
//     rollback XDP: stop bpf-sync then edge-xdp; L7 nginx continues on generational SHM caches.
//
// Invariants:
//   - Maps pinned even when XDP attach skipped (unlicensed): bpf-sync can still populate.
//   - XDP_MODE offload tries offload->native->generic; native tries native->generic.
//   - L4 drop only; rotating proxies evade host maps (see edge.mdc).
//
// Defaults and limits:
//   - INGRESS_INTERFACE (iface name, required).
//   - XDP_MODE: generic|native|offload (default generic).
//   - REDIS_ADDRS + REDIS_PASS: entitlement check via edge.EbpfEdgeLicensed (1s timeout).
//   - BPFPinDir via edge.BPFPinDir() when -pin-dir unset (/sys/fs/bpf/ad-event-processor).
//   - Pinned map count metrics ticker 30s.
//
// Verify:
// go test ./cmd/edge-xdp/ -run TestAttach_ -count=1
// go test ./internal/edge/... -count=1
package main
