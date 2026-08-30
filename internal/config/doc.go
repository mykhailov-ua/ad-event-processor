// Package config loads and normalizes process environment into the shared Config struct.
//
// Role:
//   - Load() in env.go: single entry for all binaries; calls load*Modules helpers and validateAndApplyDefaults.
//   - env_ingest.go, env_controlplane.go, env_database.go, env_edge.go, env_postback.go, env_license.go:
//     domain-specific AD_EVENT_PROCESSOR_* and legacy alias parsing.
//   - LoadLogCompactor, LoadLogEvacuator: sidecar-only config loaders (log pipeline tools).
//   - Secret type (env_parse.go): redacts DSN/password fields via slog.LogValue.
//
// Topology:
//   - Imported by cmd/* wire.go and internal/control/deps.go at process start only.
//   - pkg/naming validates product token env aliases during load.
//   - No import of internal/domain business rules; parsers map strings to typed Config fields only.
//
// Defaults and limits:
//   - ExpectedRedisShardCount 4 (static slot topology; production enforces exact match).
//   - FILTER_TIMEOUT_MS production ceiling 100 ms (dev default higher; see core.mdc).
//   - OrtbScanMaxBytes, ProtoMaxFields, HTTP1 body idle knobs bound parser budgets (parser.mdc).
//   - ManagementPort default 8188; ServerPort 8181; ProcessorPort 8186; MetricsPort 9090.
//
// Env defaults:
//   - Redis stream ad:events:stream; fraud stream ad:fraud:stream; processor group ad:processor:group.
//   - PaymentDBDSN falls back to DBDSN when unset.
//   - Boolean flags use explicit 0/1 or true/false parsing; empty means default from Load().
//   - Control module toggles under cfg.Control.Enable* (env_controlplane.go).
//
// Invariants:
//   - Load fails fast on missing DB_DSN, REDIS_ADDRS, TOKEN_SYMMETRIC_KEY, or invalid shard topology.
//   - Production: FILTER_TIMEOUT_MS <= 100; TRACKER_PG_FALLBACK must be 0; Redis shard count must match ExpectedRedisShardCount.
//   - BrokerPrimaryCH requires BROKER_URL; Redis sentinel master name count must match REDIS_ADDRS.
//   - Config is immutable after bootstrap; hot path reads snapshots not per-request Load().
//
// Forbidden:
//   - os.Getenv in request handlers (load once at main).
//   - Import internal/domain business rules into config parsers.
//
// Verify:
//
//	go test ./internal/config/ -short -count=1
//	go test ./internal/config/ -short -run TestLoad_productionFilterTimeoutCeiling -count=1
//	go test ./internal/config/ -short -run TestLoad_productionRequiresExpectedShardCount -count=1
//	go test ./internal/config/ -short -run TestValidateBrokerPrimaryRequiresBrokerURL -count=1
package config
