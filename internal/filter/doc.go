// Package filter provides Go-local gates and the FilterEngine chain ending in UnifiedFilter Redis Lua.
//
// Role:
//   - FilterEngine.Check runs synchronously on the caller goroutine (tracker: PinnedWorkerPool Tier B, not detached).
//   - Registry campaignMapSnapshot via atomic.Pointer; snapGen + per-worker one-entry cache (registryWorkerCacheSlot).
//   - UnifiedFilter in internal/filter/unified: at most one EVALSHA per accept when not local-quanta full-skip.
//
// Thread model:
//   - Tier A gnet epoll must not call Check; Tier B pinned worker runs full chain including sync EVALSHA.
//
// Topology:
//   - Subpackages: unified (Lua debit), netintel (GeoIP, LPM/CIDR, proxy/VPN feeds).
//   - SettingsWatcher fraud boost maps: snapshot readers on filter path.
//   - Tracker production chain (cmd/tracker/wire.go): license, license_rps, emergency breaker, geo, schedule,
//     vpp, fraud, residential proxy, tcp_mss, device, l7_wire, json_serialization, behavior_telemetry,
//     consent, segment, entitlements, unified (last).
//
// Invariants:
//   - Cheap filters before any Redis RTT; UnifiedFilter last.
//   - FilterDeadlineMono monotonic deadline; no wall clock in filter loops.
//   - RollbackDebit via budget-rollback.lua or local quanta refund on post-debit enqueue fail.
//   - TryReserve on stream/broker producer before Lua debit (enforced in ingest tryAcquireStreamAdmission).
//
// Forbidden:
//   - ML inference import from internal/fraud; boost snapshot only.
//   - Postgres or ClickHouse on synchronous Check path.
//
// Verify:
// go test ./internal/filter/... -short -count=1
// go test ./internal/ingest/ -short -run TestFilterEngine_TrackerSegmentAfterLocalFilters -count=1
// go test ./internal/ingest/ -short -run TestUnifiedFilter_RollbackDebit -count=1
// go test ./internal/ingest/ -short -run TestStreamProducerAdmissionRaceWithoutReserve -count=1
//
// Tradeoffs:
//   - Filter chain order (cheapest local gate first, UnifiedFilter last): license and license_rps fail
//     before GeoIP mmap; emergency breaker before netintel feeds; geo/schedule/vpp/fraud/residential/
//     tcp_mss/device/l7_wire/json_serialization/behavior_telemetry run before segment and entitlements;
//     consent before segment; unified Redis Lua last so only survivors pay EVALSHA (architecture.mdc).
//   - Edge allow-before-deny vs tracker chain: XDP allow_v4/v6 LPM and L7 generational blacklist use
//     allowlist-first / stale-stamp fail-open at the perimeter (edge_filter.c, access-check.lua); tracker
//     cannot mirror that ordering for budget debit and must fail-closed on infra (503) and on license/
//     breaker. Edge drops floods and known bad IPs early; tracker re-runs geo, fraud boost, segment, and
//     unified debit for defense in depth behind CDN termination (edge.mdc JA3/TCP disclaimers).
//   - Segment and entitlements after local intel: segment needs registry snapshot + netintel classification
//     already applied; entitlements may Redis INCR ingress RPD but stays before unified so at most one
//     budget EVALSHA remains; rejected unified before geo (every geo reject would debit-path Redis).
//   - Synchronous FilterEngine.Check on Tier B pinned worker vs detached goroutine: same goroutine holds
//     offload pin/arena lifetime and FILTER_TIMEOUT_MS monotonic deadline; rejected go func() around
//     Check (tradeoffs.mdc unsafe.String / buffer contract).
//   - Fraud boost atomic snapshot vs internal/fraud ML on Check path: snapshot read only; rejected
//     LGBM inference in filter chain (cold cmd/fraud-scorer batch).
//   - CGNAT / mobile-carrier bypass (cgnat_policy.go, entitlements ConfigureCGNAT): allow-path for shared
//     carrier NAT signals; velocity limits skipped when classified mobile carrier; not a global allow-before-
//     deny at edge, scoped to entitlements and unified TTC bypass (return 10) semantics downstream.
package filter
