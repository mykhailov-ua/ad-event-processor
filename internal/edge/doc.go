// Package edge owns XDP BPF objects, pinned map paths, Redis blocklist sync,
// TCP fingerprint staging, and ebpf_xdp_edge license gating for edge daemons.
//
// Role:
//   - bpf2go EdgeObjects loaded by cmd/edge-xdp; maps pinned under BPFPinDir.
//   - cmd/edge-bpf-sync: SyncBlocklistIncremental, SyncAllowlistFromRedis, violation/fingerprint ringbuf drains.
//   - L7 parallel path: deploy/nginx/lua/ (ngx.shared mirrors on worker 0 timers).
//     blacklist_cache: generational _bl_ver + b:{ip}; full sync sets _bl_count exactly;
//     incremental stamp_ips adds only IPs not already at current ver.
//     edge_config: generational _asn_ver + asn_* stamps; edge_config SHM 4m in nginx.conf.
//     slot_map / node_weights: metadata keys written last after table body (TOCTOU avoidance).
//     circuit_breaker: edge-circuit.lua ratio gate; writers on Redis/sync/upstream 5xx failures.
//
// Topology:
//   - Control outbox -> Redis shard 0 SET/ZSET -> edge-bpf-sync -> pinned BPF maps -> edge-xdp drop.
//   - Redis deny keys: blacklist:manual, blacklist:auto, blacklist:fraud; changelog ZSETs (48h TTL).
//   - TCP fingerprint staging: edge:tcp_fp:ip:{ip} HASH + edge:tcp_fp:recent ZSET on shard 0.
//   - Default pin dir: /sys/fs/bpf/ad-event-processor (BPF_PIN_DIR override).
//
// Defaults and limits:
//   - SYNC_INTERVAL 5s (edge-bpf-sync); blocklist full SMEMBERS interval 5 min (edge package constant).
//   - AUTOBAN_TTL 5m default; VIOLATION_POLL_INTERVAL 250ms; FINGERPRINT_POLL_INTERVAL 500ms.
//   - License check timeout 1s when REDIS_ADDRS set (EbpfEdgeLicensed).
//   - LPM blocklist_host maps max_entries=786432 (edge_filter.c; see edge.mdc).
//
// Invariants:
//   - Allow-before-deny lookup order in edge_filter.c; kernel maps read-only in XDP.
//   - Control plane never writes kernel maps directly; edge-bpf-sync is the map writer.
//   - ebpf_xdp_edge license required for sync/autoban loops; idle when unlicensed.
//   - REDIS_ADDRS unset: edge-xdp attach allowed with warn (dev only).
//
// Forbidden:
//   - internal/ingest hot path must not import this package (XDP/BPF is edge-host only).
//   - Autoban records Redis keys; does not call Postgres outbox from edge daemons.
//
// Verify:
// go test ./internal/edge/... -short -count=1
// bash scripts/test/edge/lua_tests.sh compliance
// bash scripts/ci/compliance.sh
// go test ./cmd/edge-xdp/ -run TestAttach_ -count=1
package edge
