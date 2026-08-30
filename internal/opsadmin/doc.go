// Package opsadmin serves operator ops HTTP, stack health reads, platform probes, and background scrapers.
//
// Role:
//   - HTTPHandlers under /api/v1/ops/*, /api/v1/audit/export, recon, consent, blacklist, dashboard, support bundle, RUM, ML model ops, fraud presets.
//   - RegisterOpsRoutes adds GET /health, GET /metrics, GET /ops/shards/slot-map, GET /ops/node-weights, processor weight routes (no /api/v1 prefix).
//   - ManagementOpsReader (NewReader) aggregates PG, Redis, ClickHouse, and Prometheus fan-out reads for incidents, DLQ, outbox, shards, audit export.
//   - AlertmanagerWebhook and OpsAlerter enqueue notify rows; edge metrics via FetchEdgeMetrics.
//
// Topology:
//   - Wired from controlplane adminapi_wire.go and ops_reader_bridge.go; HTTPHandlers receive Reader and Host callback deps.
//   - controlplane.Service starts MetricScraper and FilterRejectRollupWorker via http_bridge (not in this package).
//   - Domain mutations (DLQ retry, shard catchup, blacklist) delegate to controlplane Host implementations.
//
// Invariants:
//   - Ops routes use shards:read or shards:write; audit export uses audit:read; support bundle uses ops:write.
//   - DLQ legacy list and DLQ inbox are separate surfaces with distinct retry endpoints.
//   - Stack health snapshot omits secret material (StackHealthSnapshotHasSecretMaterial); outbox age drives degraded/critical status.
//   - Support bundle and webhook bodies use pkg/coldpath body limits.
//
// Forbidden:
//   - Tracker hot-path imports.
//   - Redis KEYS, FLUSHALL, or FLUSHDB from ops handlers.
//
// Verify:
//	go test ./internal/opsadmin/ -short -count=1
//	go test ./internal/opsadmin/ -short -run TestComputeStackHealthStatus -count=1
//	go test ./internal/opsadmin/ -short -run TestStackHealthSnapshot -count=1
package opsadmin
