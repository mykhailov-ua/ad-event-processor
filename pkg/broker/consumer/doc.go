// Package consumer runs a blocking fetch loop over pkg/broker/client with offset commits.
//
// Role:
//   - Consumer.Run: fetch batches, invoke Handler(payload, offset), commit via broker offset tracker.
//   - Used by cmd/processor when CH_INGEST_SOURCE=broker (shadow or live).
//
// Topology:
//   - Config: BrokerAddr, RedisURL (leader discovery), Topic, Partition, Group.
//   - Wraps client.NewClient; idle wait between empty fetches.
//
// Defaults and limits:
//   - MaxBytes default 1 MiB per fetch when zero.
//   - Timeout default 5s; IdleWait default 250ms.
//
// Forbidden:
//   - Handler errors do not advance offset (at-least-once; downstream must idempotize).
//   - Unit consumer tests alone do not prove ClickHouse ingest wiring.
//
// Verify:
// go test ./pkg/broker/consumer/... -short -count=1
// go test ./internal/ingest/ -short -run TestFault_BrokerLiveConsumer -count=1
package consumer
