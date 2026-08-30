// Package rtb implements in-process OpenRTB auction, SoA catalog, and budget authority for the tracker.
//
// Role:
//   - RunAuction and RunAuctionEval on atomic.Pointer catalog snapshots (64 geo shards).
//   - CampaignAuctionRegistry columnar slices; presorted buckets with scan budget p99 < 500 candidates.
//   - BudgetStore CAS when RTB_BUDGET_AUTHORITY=rtb; deal pacing, freq cap, creative cache SoA.
//
// Topology:
//   - Hot readers: single atomic load per auction; cold rebuild publishes new catalogSnapshot.
//   - Wired from ingest /openrtb/bid and applyRtbAuction on /track (mode-dependent); no full FilterEngine on bid path.
//   - Wire codec lives in internal/openrtb; admin deals in internal/rtbadmin (cold path).
//
// Invariants:
//   - 0 allocs/op target in RunAuction bench; fail with typed NoBidReason.
//   - Geo partition req.GeoHash & 63; StaticSlot sharding for Redis budget keys elsewhere.
//
// Forbidden:
//   - encoding/json on production /openrtb/bid response path.
//   - Postgres or outbox inside RunAuction loop.
//
// Verify:
//
//	go test ./internal/rtb/... -short -count=1
//	make test-alloc-gate
//	go test ./internal/ingest/ -short -run OpenRTB -count=1
package rtb
