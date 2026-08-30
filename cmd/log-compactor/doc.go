// Package main compacts hot log segments to warm tier (and optional cold CH rollup).
//
// Role:
//   - Load config via config.LoadLogCompactor; run logpipeline.NewCompactor on source_dir -> warm_dir.
//   - Optional S3 backend (NewS3TierStore) with local scratch for warm paths.
//   - Optional cold tier: ClickHouse rollup via logpipeline.NewColdRolluper when ColdEnabled.
//   - Optional file leader lock (LeaderElection) for single-writer compactor.
//   - Decrypt with LOG_ENCRYPTION_KEY derived key when set.
//
// Topology:
//   - Long-running daemon; metrics on 127.0.0.1:MetricsPort from config.
//   - Uses internal/logpipeline TierStore, CheckpointStore, RegisterMetrics.
//   - ClickHouse connect only when cold rollup enabled.
//
// Defaults and limits:
//   - Checkpoint files at CheckpointPath and ColdCheckpointPath resume partial work.
//   - SampleRate and HotMinAgeHours control which files are eligible.
//   - Metrics bind 127.0.0.1:MetricsPort from config.
//
// Forbidden:
//   - Not tracker ingest; no /track or Redis stream writes.
//
// Verify:
// go test ./internal/logpipeline/... -short -count=1
package main
