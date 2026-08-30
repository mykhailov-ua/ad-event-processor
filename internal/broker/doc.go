// Package broker implements the mmap WAL ingest broker gnet server and Redis-coordinated HA fencing.
//
// Role:
//   - server.go accepts pkg/broker/protocol produce/fetch over gnet; log segments via pkg/broker/log.
//   - coord.go and registry_redis.go elect leader and register topic ids; offset_redis.go tracks consumer HWM.
//   - replay.go supports cutover drills and shadow compare against Redis stream ingest.
//
// Topology:
//   - cmd/broker binary; tracker BrokerProducer is client in internal/stream/broker.
//   - Metrics wired through internal/metrics/broker_log_wire.go.
//
// Invariants:
//   - Leader fence token required before append; stale leader append rejected (fencing_test).
//   - Retention obeys segment size and age knobs; fsync policy matches durability tier tests.
//   - Ring buffer full returns ErrRingBufferFull to producer (not silent drop).
//
// Forbidden:
//   - Budget Lua or Postgres settlement inside broker server path.
//   - Unbounded in-memory topic registry without Redis backing in HA mode.
//
// Verify:
//
//	go test ./internal/broker/ -short -count=1
//	go test ./internal/broker/ -short -run TestFault_Leader -count=1
package broker
