// Package broker provides durable consumer offset persistence for mmap WAL broker clients.
//
// Role:
//   - ConsumerOffsetTracker: file-backed topic/partition/group offsets under dataDir.
//   - Used by pkg/broker/consumer and cmd/processor broker ingest paths.
//   - Wire protocol and WAL live in pkg/broker/{protocol,log,client}; server in internal/broker.
//
// Topology:
//   - Default offset dir: var/lib/ad-event-processor/broker/offsets.
//   - Offset files: {topic}_{partition}_{group}.offset (sanitized names).
//   - CommitOffset validates consumer group via pkg/broker/protocol.
//
// Defaults and limits:
//   - Empty dataDir falls back to var/lib/ad-event-processor/broker/offsets.
//   - Offset file encoding: 8-byte big-endian uint64.
//
// Forbidden:
//   - pkg/* must not import internal/* (offsets only; no controlplane admin).
//   - Not a substitute for Redis stream PEL when CH_INGEST_SOURCE=redis.
//
// Verify:
// go test ./pkg/broker/... -short -count=1
// go test ./internal/ingest/ -short -run TestFault_Broker -count=1
package broker
