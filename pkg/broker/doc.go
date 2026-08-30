// Package broker provides local consumer offset persistence for mmap WAL broker ingest.
//
// Role:
//   - ConsumerOffsetTracker: crash-safe topic/partition/group high-water marks on disk.
//   - Complements broker-server offset RPC (pkg/broker/client CommitOffset); processor
//     writes both so restarts resume from local files before querying the leader.
//   - Wire codec, mmap WAL, and TCP client live in pkg/broker/{protocol,log,client}.
//   - gnet broker daemon, HA coordinator, and server offset store live in internal/broker.
//
// pkg/internal boundary:
//   - pkg/broker/* may import pkg/* and stdlib only (no internal/*).
//   - internal/broker imports pkg/broker/{protocol,log,client} for serve path.
//   - internal/stream/broker BrokerProducer uses pkg/broker/client on tracker hot path.
//   - cmd/processor BrokerConsumerGroup uses ConsumerOffsetTracker + client directly
//     (not pkg/broker/consumer; that package is a thin reusable fetch loop for tests/tools).
//
// Offset persistence (two tiers):
//   - Local (this package): {sanitizedTopic}_{partition}_{sanitizedGroup}.offset under dataDir;
//     8-byte big-endian uint64; write temp + rename; CommitOffset is monotonic (regress ignored).
//   - Broker server (internal/broker offset_store.go): CmdCommitOffset / CmdCommittedOffset over
//     pkg/broker/protocol; MemoryOffsetStore default, RedisOffsetStore in HA; keys use
//     protocol.TopicPartitionID(topic, partition) + consumer group name.
//   - Processor startup order: read local tracker if > 0, else client.CommittedOffset RPC.
//   - After StoreBatch (or shadow count): local CommitOffset then broker CommitOffset RPC.
//
// Topology:
//   - Default offset dir when dataDir empty: var/lib/ad-event-processor/broker/offsets.
//   - CommitOffset validates group via protocol.ValidateConsumerGroup.
//   - Not a substitute for Redis stream PEL when CH_INGEST_SOURCE=redis.
//
// Invariants:
//   - Offset files use sanitized topic/group names (alnum, dash, underscore; else '_').
//   - LoadAll on NewConsumerOffsetTracker hydrates the in-memory map from *.offset files.
//
// Forbidden:
//   - pkg/* importing internal/* (including controlplane or ingest handlers).
//   - Treating local offset files alone as cross-host consumer coordination (use broker RPC).
//
// Verify:
// go test ./pkg/broker/ -short -run TestConsumerOffsetTracker -count=1
// go test ./internal/ingest/ -short -run TestFault_Broker -count=1
package broker
