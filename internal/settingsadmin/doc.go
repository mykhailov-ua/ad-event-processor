// Package settingsadmin owns system settings KV, IP blocklist, emergency breaker, and fraud-threat outbox rows.
//
// Role:
//   - Store methods called from controlplane settings and ops handlers (block/unblock IP, blacklist list,
//     settings patch, emergency breaker toggle, SyncSystemState, fraud threat enqueue).
//   - Changes enqueue outbox rows (UPDATE_BLACKLIST, UPDATE_SETTINGS) for Redis global config and blacklist fan-out.
//
// Topology:
//   - Wired via controlplane/settingsadmin_bridge.go; Host supplies Redis shard list, audit logging,
//     protected IP checks (edge allowlist), SyncGlobalConfig, SyncGlobalSetReplace, and fraud-admin hooks.
//   - Redis fan-out helpers live in internal/shardadmin; settingsadmin.Store does not dial Redis directly.
//   - blacklist_ttl.go maps manual/fraud/auto sources to TTL hours from Host config defaults.
//
// Invariants:
//   - Block IP preview path (dryRun) does not write; apply path writes PG + outbox in one transaction.
//   - Protected IPs from Host cannot be blocked.
//   - normalizeSystemSettings validates rtb_budget_authority and rtb_mode; other keys pass through unchanged.
//   - Emergency breaker and settings patches audit in the same transaction as outbox enqueue.
//
// Forbidden:
//   - KEYS/FLUSHALL on Redis from this package.
//   - Hot-path filter imports.
//
// Verify:
//
//	go test ./internal/settingsadmin/ -short -count=1
//	go test ./internal/settingsadmin/ -short -run TestResolveBlacklistExpiry -count=1
package settingsadmin
