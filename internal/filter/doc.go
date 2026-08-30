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
//
// Invariants:
//   - Cheap filters before any Redis RTT; UnifiedFilter last.
//   - FilterDeadlineMono monotonic deadline; no wall clock in filter loops.
//   - RollbackDebit via budget-rollback.lua or local quanta refund on post-debit enqueue fail.
//
// Forbidden:
//   - ML inference import from internal/fraud; boost snapshot only.
//   - Postgres or ClickHouse on synchronous Check path.
//
// Verify:
//
//	go test ./internal/filter/... -short -count=1
//	go test ./internal/ingest/ -short -run TestUnifiedFilter_RollbackDebit -count=1
//	go test ./internal/ingest/ -short -run TestStreamProducerAdmissionRaceWithoutReserve -count=1
package filter
