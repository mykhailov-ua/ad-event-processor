// Package main validates fraud ML model file and fixture feature vectors.
//
// Role:
//   - Load LightGBM model (-model); assert NFeatures matches fraud.Dims().
//   - For each features_*.json fixture: rebuild fraud.FeatureRow.ToVector() and compare to fixture.Vector within 1e-9.
//   - Exit 1 on model load, dimension mismatch, or any fixture vector drift.
//
// Topology:
//   - Offline CI/release helper; no database connections.
//   - Default paths var/fraudscore/artifacts/model.txt and var/fraudscore/fixtures.
//
// Invariants:
//   - Exit 2 on missing -model/-fixtures flag values; exit 1 on model or vector mismatch.
//
// Forbidden:
//   - Does not measure production precision/recall (no labeled holdout set in this binary).
//   - Not runtime scoring (use cmd/fraud-scorer or cmd/ml-replay).
//
// Verify:
// go run ./cmd/ml-validate
// go run ./cmd/ml-validate -model var/fraudscore/artifacts/model.txt -fixtures var/fraudscore/fixtures
package main
