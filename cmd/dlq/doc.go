// Package main is the Redis stream dead-letter queue operator CLI.
//
// Role:
//   - archive: XREAD DLQ stream, normalize to AdDLQEvent protobuf, write length-prefixed bin file, XDEL source.
//   - requeue: read DLQ stream, extract OriginalEvent, XADD to -dest target stream, XDEL from DLQ.
//   - restore: read archive file, XADD OriginalEvent as field d to -stream (default ad:events:dlq).
//   - inspect: print human-readable JSON for protobuf (AdDLQEvent/AdStreamEvent) or legacy flat hash entries.
//   - edit: XRANGE one message, $EDITOR on JSON, XDEL + XADD rewritten protobuf field d (new stream ID).
//
// Topology:
//   - Single Redis client from -redis URL (default redis://localhost:6379); no cluster routing logic.
//   - Protobuf via internal/ingest/pb; legacy flat hash fields supported on archive/inspect.
//   - No Postgres, ClickHouse, or outbox; processor idempotency is downstream responsibility.
//
// Invariants:
//   - fatal() logs JSON to stderr and os.Exit(1); missing -id on edit exits 2 (usage error).
//   - XREAD Block 10ms; empty read ends archive/requeue/inspect loops.
//   - Archive file: big-endian uint32 length + AdDLQEvent protobuf per record (O_APPEND on archive).
//   - Reuses pb.AdDLQEvent and pb.AdStreamEvent per batch row (Reset between messages).
//   - edit writes dlq-edit-*.json temp file; removed on return.
//
// Defaults and limits:
//   - -action default archive; -stream default ad:events:dlq; -dest default dlq_archive.bin.
//   - -batch default 1000 messages per XREAD.
//   - -rate 0 unlimited; >0 events/s token bucket on requeue and restore only (burst = rate).
//
// Forbidden:
//   - Not tracker /track path; no budget debit or CH sink claims from this binary.
//   - requeue/restore do not guarantee processor idempotency without downstream sync_idempotency guards.
//   - Does not enqueue controlplane outbox or mutate Postgres campaign state.
//
// Verify:
// go run ./cmd/dlq -action inspect -stream ad:events:dlq -batch 10
// go test ./cmd/dlq/... -short -count=1
// go test ./internal/ingest/pb/... -short -count=1
package main
