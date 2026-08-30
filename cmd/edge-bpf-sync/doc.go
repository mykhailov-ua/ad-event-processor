// Package main syncs Redis block/allow lists and XDP telemetry into pinned BPF maps.
//
// Role:
//   - Load pinned deny/allow/stats/violations/fingerprints maps from edge.ResolvePinnedMapPaths().
//   - Periodic SyncBlocklistIncremental and SyncAllowlistFromRedis (SYNC_INTERVAL default 5s).
//   - Drain violations ringbuf -> edge.RecordAutoBan on Redis (AUTOBAN_TTL default 5m).
//   - Drain fingerprints ringbuf -> edge.Record TCP fingerprint staging on Redis shard 0.
//   - ExportStatsToPrometheus and WriteRedis xdp stats snapshot (STATS_INTERVAL default 2s).
//   - Serve GET /metrics on METRICS_PORT default 9090.
//
// Topology:
//   - Pairs with cmd/edge-xdp; requires ebpf_xdp_edge license (idle when not licensed).
//   - Redis from edge.FirstRedisAddr(); cilium/ebpf pinned maps under /sys/fs/bpf/ad-event-processor.
//   - Parallel L7 consumer: deploy/nginx/lua/edge-blacklist-sync.lua (ngx.shared generational cache).
//   - Uses internal/edge for sync, handlers, and licensing gate.
//
// Pinned map paths (edge.ResolvePinnedMapPaths):
//   - Default pin dir: BPF_PIN_DIR or /sys/fs/bpf/ad-event-processor.
//   - Per-map overrides: BPF_BLOCKLIST_MAP, BPF_BLOCKLIST_V6_MAP, BPF_BLOCKLIST_HOST_V4_MAP,
//     BPF_BLOCKLIST_HOST_V6_MAP, BPF_ALLOWLIST_MAP, BPF_ALLOWLIST_V6_MAP, BPF_STATS_MAP,
//     BPF_VIOLATIONS_MAP, BPF_FINGERPRINTS_MAP.
//   - Files: blocklist_v4, blocklist_v6, blocklist_host_v4, blocklist_host_v6, allow_v4, allow_v6,
//     stats, violations, fingerprints (prog_array and ratelimit maps pinned by edge-xdp only).
//
// Cross-layer sync (Redis -> BPF vs Redis -> L7):
//
//	Redis SET keys blacklist:manual, blacklist:auto, blacklist:fraud (shard 0).
//	ZSET blacklist:changelog:add|remove (48h TTL; scored by Unix time; written by RecordBlacklistChangelog).
//
//	BPF path (this binary):
//	  SyncBlocklistIncremental -> every 5 min SyncBlocklistFromRedis (3x SMEMBERS + ApplyDiff)
//	  OR between full syncs: ZRangeByScore changelog delta -> ApplyHostListDelta (host maps only).
//	  BlocklistStore shadow (Go maps) tracks desired state; ebpf.Map Update/Delete on diff.
//	  Allowlist: SyncAllowlistFromRedis each SYNC_INTERVAL tick.
//
//	L7 path (nginx, not this binary):
//	  edge-blacklist-sync.lua: SMEMBERS -> stamp b:{ip}, bump _bl_ver last on full sync.
//	  Incremental quarantine: stamp_ips(..., bump_version=false); _bl_count += new IPs only.
//	  edge-config.lua: HMGET config:values -> generational _asn_ver (no dict:get_keys).
//	  edge-slot-map.lua / edge-node-weights.lua: table writes before version/peer_count metadata.
//	  edge-circuit.lua + edge-circuit-log.lua: circuit_breaker SHM err/total ratio.
//	  access-check.lua: 403 when b:{ip} == _bl_ver; stale/missing _bl_sync_ts -> 503.
//
// Goroutine model:
//   - main: single select loop on ctx.Done, syncTicker, statsTicker, violationTicker, fingerprintTicker.
//   - serveMetrics: background HTTP /metrics goroutine; shutdown via lifecycle sidecar timeout 5s.
//   - No worker pool; ringbuf drains run synchronously in main loop per tick (ViolationHandler.Drain
//     with idle window = poll interval). Post-violation autoban triggers immediate runSync when n>0.
//
// Fail-open vs fail-closed at startup:
//   - Fail-closed (os.Exit 1): deny maps LoadPinnedBlocklist*, allow maps LoadPinnedAllowlist*,
//     rlimit.RemoveMemlock, missing REDIS_ADDRS.
//   - Fail-open (Warn, continue): stats map missing (XDP metrics disabled); violations ringbuf
//     missing (autoban disabled); fingerprints ringbuf missing (IVT staging disabled).
//   - Licensed idle: EbpfEdgeLicensed false -> skip sync/autoban/drain ticks; pinned maps still opened.
//
// Cache invalidation labels:
//   - bpf_full_resync: needsFullSync (5 min) -> SMEMBERS all deny sets -> ApplyDiff -> explicit Delete on removed hosts/prefixes.
//   - bpf_changelog_delta: ZSET delta since lastScore -> ApplyHostListDelta add/remove with Delete.
//   - bpf_lru_implicit: blocklist_host_* full -> kernel LRU evict; shadow may drift until bpf_full_resync.
//   - l7_generational_full: nginx sync() bumps _bl_ver; old b:{ip} entries become non-blocking.
//   - l7_generational_incremental: stamp_ips without bump; new blocks only; _bl_count deduped;
//     unblocked IPs not deleted from SHM (stale stamp fail-open).
//   - violation_autoban: ringbuf drain -> RecordAutoBan -> Redis SADD + changelog -> runSync.
//
// Defaults and limits:
//   - SYNC_INTERVAL 5s; STATS_INTERVAL 2s; VIOLATION_POLL_INTERVAL 250ms; FINGERPRINT_POLL_INTERVAL 500ms.
//   - AUTOBAN_TTL 5m; METRICS_PORT default 9090; metrics shutdown timeout 5s.
//   - Blocklist full SMEMBERS interval 5 min (edge package constant).
//
// Invariants:
//   - Control plane never writes kernel maps directly; this daemon is the only map writer.
//   - ebpf_xdp_edge license required for sync/autoban loops; idle warn when unlicensed.
//   - Missing stats/violations/fingerprints maps degrade gracefully (warn, continue).
//   - rlimit.RemoveMemlock required at startup.
//
// Forbidden:
//   - Not imported by tracker or internal/ingest hot path.
//   - Autoban records Redis keys; does not call Postgres outbox from this binary.
//
// Verify:
// go test ./internal/edge/... -short -count=1
// bash scripts/test/edge/lua_tests.sh compliance
// bash scripts/ci/compliance.sh
package main
