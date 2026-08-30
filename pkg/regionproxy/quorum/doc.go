// Package quorum implements Redis-backed operation lease book/ack for multi-region batches.
//
// Role:
//   - Book, Ack, Status for operation IDs across replica node list.
//   - Lease keys: ad_event_processor:op:lease:{hex}; ack subkeys per node.
//   - Required quorum default 2 when replicaCount > 1.
//
// Topology:
//   - Called from internal/controlplane operation_lease paths when Enterprise multi_region enabled.
//   - Redis shard 0; pairs with pkg/regionproxy/uplink global ingest batches.
//
// Defaults and limits:
//   - leaseTTL 48h; states booked/executing/completed.
//   - Required(1) returns quorum 1 for single-node cells.
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy.
//
// Verify:
// go test ./pkg/regionproxy/quorum/... -short -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package quorum
