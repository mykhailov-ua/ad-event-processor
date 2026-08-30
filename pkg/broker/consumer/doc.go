// Package consumer runs a blocking fetch loop with broker RPC offset commits.
//
// Role:
//   - Consumer.Run: poll Fetch, invoke Handler(payload, offset), CommitOffset on success.
//   - Thin library over pkg/broker/client; offsets persist on broker leader (not local files).
//   - Used in internal/broker offset/resume tests; production CH ingest uses
//     cmd/processor BrokerConsumerGroup (local ConsumerOffsetTracker + client RPC).
//
// pkg/internal boundary:
//   - No internal/* imports; server and offset store are internal/broker.
//   - For dual local+broker offset persistence see pkg/broker ConsumerOffsetTracker.
//
// Topology:
//   - Config: BrokerAddr, optional RedisURL (leader failover), Topic, Partition, Group.
//   - Startup: client.CommittedOffset; loop Fetch from committed high-water mark.
//   - Handler error: commit prior contiguous offsets, return error (at-least-once).
//   - RedisURL set: fetch/commit errors sleep IdleWait and continue (failover tolerant).
//
// Defaults and limits:
//   - MaxBytes default 1 MiB per fetch when zero.
//   - Timeout default 5s; IdleWait default 250ms between empty fetches or retry waits.
//
// Invariants:
//   - nextCommit = message offset + 1 (consumer group offset is exclusive end).
//   - Handler errors do not advance offset for the failing message; downstream must idempotize.
//
// Forbidden:
//   - Unit consumer loop alone as ClickHouse ingest wiring proof.
//   - Expecting pkg/broker ConsumerOffsetTracker updates (this package does not call it).
//
// Verify:
// go test ./internal/broker/ -short -run TestConsumerRunAndResume -count=1
package consumer
