// Package main uploads local log segments to S3 (log evacuator).
//
// Role:
//   - Scan LogDir on ScanIntervalMs; upload closed segments via logpipeline.NewS3Store.
//   - Persist progress in CheckpointPath; optional RequireCompactorMarker gate before upload.
//   - Multipart threshold from config for large objects.
//
// Topology:
//   - Node-local I/O sidecar near log files; tools compose profile.
//   - Config from config.LoadLogEvacuator; no Postgres or tracker wiring.
//
// Invariants:
//   - Shutdown on context cancel; non-cancel errors exit 1.
//   - S3 client built once at startup from region, bucket, prefix, endpoint flags in config.
//
// Forbidden:
//   - Does not parse /track payloads or write ClickHouse directly.
//   - Not a disk-cache eviction daemon for unrelated volumes.
//
// Verify:
// go test ./internal/logpipeline/... -short -count=1
package main
