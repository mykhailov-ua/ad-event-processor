// Package main is the mmap WAL ingest broker daemon and replay CLI.
//
// Role:
//   - Default: broker serve (gnet) with WAL under -data-dir; HA via -redis-url coordinator.
//   - broker replay: WAL segment replay utility (see replay.go).
//   - --health-probe <url>: lifecycle health probe exit code for compose.
//   - Primary ingest path when CH_INGEST_SOURCE=broker; tracker StreamProducer/BrokerProducer clients.
//
// Topology:
//   - Listen default 127.0.0.1:9092 or runtimepaths.BrokerGnetSocket unix path.
//   - Health default 127.0.0.1:8084 or runtimepaths.BrokerHealthSocket.
//   - pkg/broker/{protocol,log,client} wire codec; internal/broker coordinator in serve.go.
//   - Shadow mode at consumer (BROKER_SHADOW_MODE); cutover per tradeoffs.mdc.
//
// Defaults and limits:
//   - -data-dir default /var/lib/ad-event-processor/broker.
//   - -max-seg-mb default 64; -index-kb default 4.
//   - Redis URL from -redis-url, BROKER_REDIS_URL, or first REDIS_ADDRS entry.
//
// Invariants:
//   - Sequential WAL segment append; segment size/index interval from serve flags.
//   - Health probe and serve share lifecycle patterns with other daemons.
//
// Forbidden:
//   - Claiming broker-primary safety from unit mockBrokerClient tests alone (need fault tier).
//   - Claiming zero-alloc or broker-primary safety from comments alone; use bench and fault tier tests.
//
// Verify:
// go test ./pkg/broker/... -short -count=1
// go test ./internal/ingest/ -short -run TestFault_Broker -count=1
package main
