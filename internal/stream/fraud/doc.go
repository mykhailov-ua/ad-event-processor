// Package fraud implements the hot-path fraud analytical stream writer and aggregate counters.
//
// Role:
//   - fraud_stream_queue.go: FraudStreamWriter MPSC analytical ring (3584 slots) plus critical lane (512 slots).
//   - fraud_stream_aggregate.go: subnet aggregation table (4096 cells) when ring fill exceeds ~80% or force-aggregate on.
//   - fraud_backpressure.go: StartFraudBackpressureWatcher reads fraud:agg_force and publishes consumer PEL lag metrics.
//   - fraud_stream_metrics.go: ring fill, mode, pending, and drop counters for Grafana parity with ClickHouse funnels.
//   - Optional broker sink via SetBrokerSink (internal/stream/broker.FraudBrokerSink) skips Redis XADD when CH_INGEST_SOURCE=broker.
//
// Topology:
//   - Called from tracker Tier B after filter decision; background worker flushes XADD batches to per-shard Redis streams.
//   - L3 blocklist and dual-L1-high signals bypass aggregation (fraudAggregateExempt); L3 never coalesced (holdout).
//   - layer_desync_count copied into vtproto payload for CH funnel parity with edge vs filter desync signals.
//
// Invariants:
//   - fraudAggregateExempt events (L3 blocklist, >=2 L1-high signals) route to critical lane only; never enter agg table.
//   - Analytical lane: default path; ring full at >=80% fill switches to aggregate mode or increments ad_fraud_stream_dropped_total.
//   - Critical lane: smaller ring (512) with spin cap (32); hard rejects must not be silently dropped when analytical ring is full.
//   - Aggregate mode emits fraud_aggregate event type with subnet/reason counts instead of per-event rows.
//   - StartFraudBackpressureWatcher toggles SetForceAggregate from Redis fraud:agg_force when consumer lag exceeds threshold.
//
// Tradeoffs:
//   - Dual rings (3584 analytical + 512 critical): analytical ring absorbs volume with optional subnet aggregation under backpressure;
//     critical lane preserves per-event rows for enforcement signals that must not be coalesced. Rejected: single ring with drop-on-full
//     for L3 blocks (TestFault_FraudStreamL3NeverAggregated holdout).
//   - L3 exemption: fraudAggregateExempt sends L3 blocklist and dual-L1-high events to critical lane even when analytical fill >=80%;
//     aggregation subnet table skips exempt codes. CH funnels count individual blocks; coalescing would under-count enforcement.
//   - Aggregation vs per-event: at fraudAggThreshold (~80% of analytical usable slots) or fraud:agg_force, subnet/reason cells collapse
//     many signals into fraud_aggregate rows to protect Redis stream RAM and processor PEL. L2-only and single-L1 signals may aggregate.
//   - Broker vs Redis sink: SetBrokerSink skips shard XADD when CH_INGEST_SOURCE=broker; same slot layout and exempt routing either way.
//
// Forbidden:
//   - internal/fraud ML inference on synchronous /track path (batch scorer cmd/fraud-scorer only).
//   - Dropping hard fraud rejects without stream row when funnels count blocks.
//   - Aggregating L3 blocklist events (TestFault_FraudStreamL3NeverAggregated holdout).
//
// Verify:
//
//	go test ./internal/stream/fraud/ -short -count=1
//	go test ./internal/stream/fraud/ -short -run TestFault_FraudStreamL3NeverAggregated -count=1
//	go test ./internal/stream/fraud/ -short -run TestFraudStreamWriter_ringFullIncrementsDropMetric -count=1
//	go test ./internal/ingest/ -short -run TestGnetHandler_fraudStreamWriteError -count=1
package fraud
