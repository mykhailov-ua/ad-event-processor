// Package uplink forwards committed regional WAL batches to global control ingest HTTP.
//
// Role:
//   - Worker polls WAL via opkey.Pool; POST JSON batches to GLOBAL_INGEST_URL with API key.
//   - BatchCommitter coordinates quorum ack after successful global ingest.
//   - Retry with ForwardMaxAttempts and ForwardRetryBackoff on transient HTTP errors.
//
// Topology:
//   - Wired from cmd/region-proxy when -global-ingest-url set.
//   - Depends on pkg/regionproxy/opkey pool and pkg/regionproxy/wal.
//
// Defaults and limits:
//   - PollInterval default 1ms when unset; BatchSize default 64.
//   - HTTPTimeout configurable per uplink.Config.
//
// Forbidden:
//   - Not tracker /track listener; regional processor uses separate MULTI_REGION_ENABLED path.
//
// Verify:
// go test ./pkg/regionproxy/uplink/... -short -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package uplink
