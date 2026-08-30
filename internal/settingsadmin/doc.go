// Package settingsadmin owns system settings KV, IP blocklist, and emergency breaker mutations.
//
// Role:
//   - Store methods called from controlplane settings handlers (blacklist TTL, block IP, emergency breaker, settings patch).
//   - Changes enqueue outbox rows for Redis global config and blacklist fan-out to all shards.
//
// Topology:
//   - Wired via settings bridge; Host supplies Redis shard list, audit logging, protected IP checks, and SyncGlobalSetReplace.
//   - blacklist_ttl.go maps manual/fraud/auto sources to TTL hours from config defaults.
//
// Invariants:
//   - Block IP preview path does not write; apply path writes PG + outbox in one transaction.
//   - Protected IPs from Host cannot be blocked.
//   - Settings normalize strips unknown keys before outbox marshal.
//
// Forbidden:
//   - KEYS/FLUSHALL on Redis from this package.
//   - Hot-path filter imports.
//
// Verify:
//
//	go test ./internal/settingsadmin/ -short -count=1
//	go test ./internal/settingsadmin/ -short -run TestBlacklistTTL -count=1
package settingsadmin
