// Package features defines fraud ML feature vectors, residential-intel enrichment, and ClickHouse PII hashing helpers.
//
// Role:
//   - features.go builds a 16-dim vector from FeatureRow; feature_spec.go lists FeatureNames and Dims().
//   - residential_intel_* files resolve proxy/VPN signals via HTTP provider, Redis feed, and residential_intel_cache CH inserts.
//   - clickhouse_pii_hasher.go hashes IPs for CH queries via pkg/piihash (SetPIIHasher for production wiring).
//   - Pure transforms and enrichment ticks; no HTTP routes in this package.
//
// Topology:
//   - Cold path only; consumed by internal/fraud/scorer, internal/fraud export aliases, and cmd/fraud-scorer sidecar.
//   - Scores publish async to Redis ml:score:boost:* via scorer.MicroBatcher, not from this package directly.
//
// Invariants:
//   - FeatureNames order and len(FeatureNames) match scorer Dims() and FeatureRow.ToVectorInto buffer size.
//   - Residential intel enricher skips invalid IPs; CH insert uses hashed ip_hash column, not raw IP strings.
//
// Forbidden:
//   - Import on tracker /track synchronous path.
//
// Verify:
//
//	go test ./internal/fraud/features/ -short -count=1
//	go test ./internal/fraud/features/ -short -run TestResidentialIntelResult_holdoutWithoutFarmHeuristic -count=1
package features
