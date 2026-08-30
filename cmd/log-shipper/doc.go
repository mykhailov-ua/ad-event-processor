// Package main tails length-prefixed active.log and produces to the mmap WAL broker.
//
// Role:
//   - Read 4-byte big-endian length + payload frames from -log-file (default /var/log/ad-event-processor/active.log).
//   - Fan out to -workers goroutines; each uses pkg/broker/client Produce on -topic (default tracker-logs).
//   - Optional Redis URL on client for broker leader discovery (-redis-url).
//   - Detect log rotation (file size shrink) and reopen active path.
//
// Topology:
//   - Co-located with tracker log writer; broker default 127.0.0.1:9092.
//   - Jobs channel capacity 10000; payload copied per frame before enqueue.
//
// Defaults and limits:
//   - Initial payload buffer 1 MiB; grows per oversized frame.
//   - EOF poll sleep 5ms; read error backoff 1s; worker progress log every 5s.
//   - Worker broker reconnect backoff 1s.
//
// Invariants:
//   - Shutdown closes jobs channel and waits with lifecycle.TimeoutsFromEnv().Wait.
//
// Forbidden:
//   - Does not write ClickHouse or Redis streams directly (broker consumer path only).
//   - Not on /track request path.
//
// Verify:
// go test ./pkg/broker/... -short -count=1
package main
