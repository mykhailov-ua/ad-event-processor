// Package entitlements maps JWT claims to feature flags, ingest gates, tier limits, and license epoch pubsub.
//
// Role:
//   - effective.go merges deployment and customer limits; ingest_gate.go answers tracker license filter snapshot.
//   - license_epoch_pubsub.go broadcasts epoch bumps to Redis; heartbeat_policy.go enforces vendor heartbeat when enabled.
//
// Topology:
//   - Called from control bootstrap and licensingadmin gate; tracker EntitlementsFilter reads Redis INCR snapshot path.
//
// Invariants:
//   - Customer limits never exceed deployment ceiling (P-C4-03 license verify catalog).
//   - FeatureAllowed checks SKU yaml tier_policy tables.
//
// Forbidden:
//   - Per-request PG fetch for entitlements on hot path.
//
// Verify:
//
//	go test ./internal/licensing/entitlements/... -short -count=1
package entitlements
