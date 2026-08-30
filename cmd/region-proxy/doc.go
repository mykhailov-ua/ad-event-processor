// Package main is the multi-region WAL ingress proxy (license feature multi_region).
//
// Role:
//   - gnet server on -addr with mmap WAL under -data-dir.
//   - Assign regional operation keys (keygen, opkey) and optional uplink batch to GLOBAL_INGEST_URL.
//   - broker.NewCoordinator for Redis HA leader election; health/metrics on -health-addr.
//   - Ready probe pings -redis-url before accepting traffic.
//
// Topology:
//   - License JWT features.multi_region; not default single_vps appliance SKU.
//   - internal/regionproxy server + pkg/regionproxy/{wal,quorum,keygen,opkey,uplink}.
//   - Regional processor flushes spend batches when MULTI_REGION_ENABLED=1.
//   - iogate.DiskWriteGate backs pressure-aware disk writes.
//
// Invariants:
//   - Hot ingestion (internal/ingest) must not import pkg/regionproxy.
//   - WAL append-only with group commit; torn records discarded on Recover().
//   - Not a substitute for Postgres balance_ledger authority (global control reconciles spend).
//   - Not on tracker /track listener ports 8181-8184.
//
// Defaults and limits:
//   - -addr default runtimepaths.RegionProxyGnetSocket() (unix under run dir).
//   - -health-addr default runtimepaths.RegionProxyHealthSocket().
//   - -data-dir default /tmp/ad-event-processor-region-proxy.
//   - -node-id default region-proxy-1; -region-code default 1.
//   - keygen/opkey PollInterval 1ms, BatchSize 256; opkey Watermark 1000.
//   - uplink PollInterval 1ms, BatchSize 64 when -global-ingest-url set.
//   - GLOBAL_INGEST_URL and GLOBAL_INGEST_API_KEY env for uplink when URL non-empty.
//   - -redis-url default redis://127.0.0.1:6379/0 (coordinator + ready probe).
//
// Verify:
// go test ./pkg/regionproxy/... -short -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package main
