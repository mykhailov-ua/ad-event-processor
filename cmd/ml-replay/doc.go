// Package main replays fraud feature rows through a LightGBM model offline.
//
// Role:
//   - Load model (-model, default var/fraudscore/artifacts/model.txt); require scorer.Dims() == fraud.Dims().
//   - Score fixtures (var/fraudscore/fixtures/features_*.json) or ClickHouse ml_features_1m (-clickhouse, -limit, -minutes).
//   - Emit CSV: feature columns, ml_score, fraud_score, tier, shadow action (boost/blacklist/silent_reject).
//   - Uses fraud.DecideWithCampaign and shadowAction helpers (no live outbox enqueue).
//
// Topology:
//   - Offline CLI; optional CH readonly via config.Load and database.ConnectClickHouseReadonly.
//   - internal/fraud LGBMScorer and feature vectorization only.
//
// Invariants:
//   - Model NFeatures must equal fraud.Dims(); exit 2 on flag parse errors, 1 on run errors.
//   - shadowAction uses silent_reject_enabled=true for IVT tier (offline policy preview only).
//
// Forbidden:
//   - Not a training pipeline; does not write Redis boost keys or enqueue outbox events.
//   - Not on tracker hot path.
//
// Verify:
// go run ./cmd/ml-replay -fixtures var/fraudscore/fixtures
// go test ./internal/fraud/... -short -count=1
package main
