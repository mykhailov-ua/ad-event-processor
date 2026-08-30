// Package rtbadmin serves RTB deal CRUD, floor optimizer admin, and RTB operator HTTP routes.
//
// Role:
//   - HTTPHandlers under /api/v1/rtb/*: deals, validate-bid-request, integration-profile,
//     shadow-diff, reconcile/export (CH stats + optional live-gate fields).
//   - FloorsHTTPHandlers: POST /api/v1/rtb/floors/apply (dry_run query supported).
//   - RunFloorOptimizer worker tick (floors.go) and FloorOptimizerWorker interval loop (worker_floor.go).
//   - SimulateRtbBidShade service method; Host injects tracker-side RunRtbBidShadeSim from control bootstrap.
//
// Topology:
//   - Wired via controlplane/rtb_bridge.go; Service implements rtbadmin.Host and FloorsHost.
//   - Deal and floor mutations enqueue RELOAD_RTB_CATALOG outbox in the same PG transaction as the write.
//   - adminapi_wire_domains.go registers RtbHTTP and RtbFloorsHTTP; ReconcileCH reads ClickHouse via Service.
//   - ShadowDiff and LiveGate handler fields are optional injectors (in-memory rtb shadow buckets when wired);
//     default control wiring leaves them nil and endpoints return source=unavailable unless extended.
//   - Tracker hot auction stays in internal/rtb; this package is cold-path admin only.
//
// Invariants:
//   - Deal mutations audit with actor id; catalog reload outbox uses distinct trigger strings per mutation.
//   - License feature gate openrtb on HTTPHandlers (licensingadmin.RequireLicenseFeature).
//   - Floors apply uses settings:write permission; deal routes use rtb:read / rtb:write.
//   - Floor optimizer writes rtb_floor_suggestions PG rows and may enqueue catalog reload on apply.
//
// Forbidden:
//   - RunAuction or FilterEngine on admin HTTP path.
//   - Import into internal/ingest hot OpenRTB handler except via published catalog snapshots.
//
// Verify:
//
//	go test ./internal/rtbadmin/ -short -count=1
//	go test ./internal/rtbadmin/ -short -run TestBestFloorBucketByPlacement -count=1
package rtbadmin
