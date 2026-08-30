// Package keygen assigns regional dedup keys to WAL records for uplink idempotency.
//
// Role:
//   - KeyGen polls WAL tail; stamps dedupkey-derived IDs using region code and node ID.
//   - BatchSize and PollInterval tune background goroutine throughput.
//
// Topology:
//   - Started from internal/regionproxy server SetKeyGen; cmd/region-proxy -region-code flag.
//   - Uses pkg/dedupkey and pkg/regionproxy/wal scan APIs.
//
// Defaults and limits:
//   - PollInterval default 1ms; BatchSize default 64 when zero in New.
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy.
//
// Verify:
// go test ./pkg/regionproxy/keygen/... -short -count=1
package keygen
