// Package fraud implements the hot-path fraud analytical stream writer and aggregate counters.
//
// Role:
//   - FraudStreamWriter enqueues fraud signals to Redis stream or broker sink after filter outcomes.
//   - MPSC rings for analytical vs critical lanes; aggregate mode at configurable threshold.
//   - Layer-desync and backpressure metrics for fraud funnel parity with ClickHouse.
//
// Topology:
//   - Called from Tier B worker after filter decision; flush batches on dedicated goroutine.
//   - Broker sink optional via internal/stream/broker/fraud_broker_sink.
//
// Forbidden:
//   - internal/fraud ML inference on synchronous /track path (batch scorer only).
//   - Dropping hard fraud rejects without stream row when funnels count blocks.
//
// Verify:
//
//	go test ./internal/stream/fraud/... -short -count=1
//	go test ./internal/ingest/ -short -run TestGnetHandler_fraudStreamWriteError -count=1
package fraud
