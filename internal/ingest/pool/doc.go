// Package pool hosts RCU domain-pool snapshots for tracking-host rotation.
//
// Role:
//   - Table maps pool_id to active/banned hostnames for click landing, DMR, and safe-page rotation.
//   - Sync polls domain_pool_domains from Postgres on interval and publishes atomic generations.
//   - BuildSnapshotFromRows and BuildTrackingDomainRotation assemble redirect targets from snapshots.
//
// Topology:
//   - Sync.Start background goroutine (cmd/tracker wire); hot path reads Table via atomic.Pointer load only.
//   - ConfigureDomainPool on AdsPacketHandler wires Table into /click and landing host pickers.
//
// Thread model (hot-path.mdc Tracker thread model):
//
//	Tier B (PinnedWorkerPool): FallbackHost and rotation helpers run synchronously during /click React
//	  on the pinned worker; zero locks, single atomic.Pointer snapshot load per lookup.
//	Background Sync goroutine: Postgres query + Publish; never blocks Tier A epoll or Tier B accept path.
//
// Invariants:
//   - Per-request Postgres query forbidden on /track or /click accept path.
//   - Banned host triggers FallbackHost scan within the same pool snapshot (active successor or wrap).
//
// Forbidden:
//   - Blocking on Sync reload from FilterEngine.Check, processTrack, or Tier A gnet thread.
//
// Verify (tests live in parent internal/ingest; subpackage has no *_test.go):
//
//	go test ./internal/ingest/ -short -run TestDomainPoolTable_FallbackHost -count=1
//	go test ./internal/ingest/ -short -run TestClickRedirect_DomainRotation -count=1
//	go test ./internal/ingest/ -short -run TestBuildTrackingDomainRotation -count=1
//
// Tradeoffs:
//   - RCU atomic.Pointer snapshot vs per-request Postgres domain_pool_domains query: snapshot chosen;
//     PG on /click accept path rejected (SLA + hot-path.mdc Tier B must not block on admin DB).
//   - Background Sync poll interval vs freshness: stale pool for one interval is acceptable; landing
//     rotation is best-effort and reconciled on next publish; rejected blocking Tier B on reload.
//   - In-snapshot banned-host linear scan vs Redis SET membership on every redirect: scan stays in
//     process memory on the pinned worker (~tens of hosts per pool); rejected extra Redis RTT on /click.
//   - Pool-scoped fallback (next active host in same pool_id) vs global domain ban table: pool rotation
//     isolates blast radius; global bans remain on edge L7/XDP and fraud filters, not this table.
//   - Sync goroutine isolated from hot path vs inline refresh on cache miss: miss does not trigger PG;
//     rejected LISTEN/NOTIFY or admin webhook into tracker for domain edits (cold outbox -> PG -> poll).
package pool
