// Package config loads and normalizes process environment into the shared Config struct.
//
// Role:
//   - env.go and env_controlplane.go parse AD_EVENT_PROCESSOR_* knobs for tracker, control, processor, and sidecars.
//   - Secret type redacts DSN and password fields in String() for logs.
//
// Topology:
//   - Imported by cmd/* wire.go and internal/control/deps.go at process start only.
//   - pkg/naming validates product token env aliases during load.
//
// Defaults and limits:
//   - ExpectedRedisShardCount 4 (static slot topology).
//   - FILTER_TIMEOUT_MS production ceiling 100 ms (dev default higher; see core.mdc).
//   - OrtbScanMaxBytes, ProtoMaxFields, and HTTP1 body idle knobs bound parser budgets (parser.mdc).
//
// Env defaults:
//   - ServerPort, ManagementPort, MetricsPort, RedisAddrs, DBDSN loaded from environment with documented fallbacks in env.go.
//   - Boolean flags use explicit 0/1 or true/false parsing; empty means default from Load().
//
// Invariants:
//   - Load fails fast on invalid Redis addr list or shard count mismatch.
//   - Config is immutable after bootstrap; hot path reads snapshots not per-request Load().
//
// Forbidden:
//   - os.Getenv in request handlers (load once at main).
//   - Import internal/domain business rules into config parsers.
//
// Verify:
//
//	go test ./internal/config/ -short -count=1
//	go test ./internal/config/ -short -run TestEnvLoad -count=1
package config
