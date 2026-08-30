// Package scorer implements LGBM, ONNX, and ensemble microbatch scorers for cmd/fraud-scorer sidecar.
//
// Role:
//   - lgbm_native_scorer.go and scorer.go load model files; microbatch.go aggregates domain.Event rows and writes Redis boosts.
//   - ensemble.go combines scorer outputs; onnx build tag selects real vs stub ONNX scorer.
//   - microbatch_config.go sets flush interval (default 50ms) and MaxStreamLagSec pause threshold (default 30s).
//
// Topology:
//   - Invoked from cmd/fraud-scorer MicroBatcher loop and fraudadmin FraudExplainScorer when model path configured.
//   - Never linked into tracker binary hot path.
//
// Invariants:
//   - eventsChan capacity 10000; each flush drains up to 10000 aggregated keys before ScoreBatch.
//   - Scoring pauses when processor stream lag exceeds MaxStreamLagSec; paused mode refreshes ml:score:boost TTL only.
//   - Suspect-tier boosts use ScoreBoostTTL (900s) on ml:score:boost:{campaign_id} shard keys.
//
// Forbidden:
//   - Synchronous ScoreBatch from FilterEngine.Check.
//
// Verify:
//
//	go test ./internal/fraud/scorer/ -short -count=1
//	go test ./internal/fraud/scorer/ -short -run TestMicroBatch_AggregationAndScoring -count=1
//	go test ./internal/fraud/scorer/ -short -run TestMicroBatch_StreamLagPause -count=1
package scorer
