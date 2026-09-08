// Package probecluster derives stable probe cluster IDs from fingerprint tuples
// and evaluates Redis-backed session cardinality for coordinated reconnaissance.
//
// Role:
// - Pure tuple layout, HMAC cluster_id, sandbox routing policy.
// - Redis I/O lives in internal/filter ProbeClusterStore.
//
// Verify:
// go test ./pkg/probecluster/ -short -run Cluster -count=1
// go test ./internal/filter/ -short -run ProbeCluster -count=1
package probecluster
