// Package rtbadmin serves RTB deal CRUD, floor optimizer admin, shadow/live gate reads, and bid-shade simulation.
//
// Role:
//   - HTTP under /api/v1/rtb/* (deals, floors, shadow diff, live gate, reconcile hints).
//   - worker_floor.go runs floor optimizer tick; floors_handlers.go exposes floor CRUD separate from deal handlers.
//
// Topology:
//   - Wired via rtbadmin_bridge.go; Host enqueues rtb catalog reload outbox after deal mutations.
//   - SimulateBidShade delegates to domain.RtbBidShadeInput on controlplane Host; tracker uses internal/rtb auction only.
//
// Invariants:
//   - Deal mutations audit with actor id; catalog reload outbox idempotent per trigger string.
//   - License feature gate required for OpenRTB admin surfaces when SKU disables RTB.
//   - Shadow diff and live gate readers are read-only CH/Prometheus aggregations.
//
// Forbidden:
//   - RunAuction or FilterEngine on admin HTTP path.
//   - Import into internal/ingest hot OpenRTB handler except via published catalog snapshots.
//
// Verify:
//
//	go test ./internal/rtbadmin/ -short -count=1
//	go test ./internal/rtbadmin/ -short -run TestFloors -count=1
package rtbadmin
