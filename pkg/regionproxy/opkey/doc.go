// Package opkey pools operation keys with MPSC queue and WAL watermark backpressure.
//
// Role:
//   - Pool enqueues op keys from ingress; BatchCommitter marks quorum-complete batches.
//   - MPSCQueue power-of-two size default 4096; Watermark default 1000 shed threshold.
//   - slot.go defines per-partition key slots written back into WAL.
//
// Topology:
//   - Wired from cmd/region-proxy SetOpKey; uplink.Worker uses BatchCommitter for ack path.
//   - PollInterval default 1ms; BatchSize default 64.
//
// Defaults and limits:
//   - QueueSize must be power of two; invalid values reset to 4096.
//   - shedTotal metric when watermark exceeded (load shedding).
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy.
//
// Verify:
// go test ./pkg/regionproxy/opkey/... -short -count=1
package opkey
