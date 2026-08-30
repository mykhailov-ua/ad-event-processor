// Package testutil provides testcontainer helpers, migration path helpers, and ingest test wiring for integration tests.
//
// Role:
//   - postgres.go / redis.go: SetupPostgres, SetupRedis, SetupAdsPostgres, SetupInjectionEnv (PG + Redis pair).
//   - paths.go: ModuleRoot and per-service migrations dirs (Ads, Payment, Billing, Notify, ServiceMigrationsDir).
//   - registry.go / mock_registry.go / registry_watch.go: NewAdsRegistry and campaign registry test helpers.
//   - shard.go: CampaignIDForShard for static-slot shard tests.
//   - injection_env.go: NewInjectionEnv bundles PG + Redis containers for fault/resilience tests.
//   - emergency_breaker.go / filter.go: SettingsWatcher and EmergencyBreakerFilter constructors for ingest tests.
//   - fault_proof.go: LogFaultProof wrapper for pkg/faultproof telemetry in tests.
//   - redis_fault.go, cohort_registry.go: Redis fault injection and cohort registry helpers.
//
// Topology:
//   - Test-only package; imported from *_test.go, tests/*, and *_integration_test.go (not production cmd/*).
//   - No ClickHouse testcontainer helper in this package (consumers use database or bespoke CH setup).
//
// Invariants:
//   - Setup helpers return cleanup func; callers must defer cleanup to avoid container leaks.
//   - Integration tests skip with integration: prefix in -short mode (anti_slop_gate.sh).
//   - SetupPostgres runs sql migrations from cfg.MigrationDirs before returning the pool.
//
// Forbidden:
//   - Production import from cmd/* main or hot-path non-_test.go packages.
//
// Verify (no package _test.go; exercised via consumers):
//
//	go list -e ./internal/testutil/...
//	go test ./internal/controlplane/ -run TestIsTLSAllowed_roles -count=1
//	make test-integration
package testutil
