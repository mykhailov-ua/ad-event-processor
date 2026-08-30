// Package rtb implements in-process OpenRTB auction, SoA catalog snapshots, and RTB budget authority on the tracker.
//
// Role:
//   - Registry.RunAuction / RunAuctionEval on atomic.Pointer catalogSnapshot (64 geo shards, geoShardMask=63).
//   - CampaignAuctionRegistry columnar slices; geo/target buckets presorted; creativeCacheSoA for multi-creative winners.
//   - BudgetStore.CheckAndSpendAll CAS in auction loop when ingest sets RTB_BUDGET_AUTHORITY=rtb.
//   - catalog_glue_*: RtbCatalog reload, deal floors, pacing, freq cap, budget mirror/reconcile workers.
//   - internal/openrtb owns wire codec; internal/rtbadmin owns cold-path deal CRUD.
//
// Topology:
//   - Hot readers: single atomic.Pointer load per auction; cold rebuild publishes immutable catalogSnapshot.
//   - Wired from internal/ingest applyRtbAuction on /track and runOpenRTBExchangeParsed on POST /openrtb/bid.
//   - No full FilterEngine chain on /openrtb/bid (dedicated exchange path per architecture.mdc).
//
// Defaults and limits (hot-path SLA; core.mdc, rtb.mdc):
//   - RunAuction p99 target < 15 us (exchange microbench tier; not full ingress E2E).
//   - rankMaxScanCandidates = 500 hard cap per auction (NoBidScanLimit beyond cap).
//   - BenchmarkAuction / make test-alloc-gate: 0 allocs/op target on RunAuction bench.
//   - Geo partition: req.GeoHash & 63; StaticSlot sharding for Redis budget keys elsewhere.
//
// Modes (RTB_MODE env via RtbModeFromConfig / RtbModeFromSetting):
//   - off: auction disabled.
//   - shadow: RunAuctionEval only on /track; recordRtbShadowAuction metrics; no campaign rewrite or spend.
//   - live: RunAuction with spend when authority allows; may rewrite campaign_id on /track when winner found.
//
// Budget authority (RTB_BUDGET_AUTHORITY env):
//   - rtb: CheckAndSpendAll in RunAuction; RtbBudgetMirrorWriter async Redis debit; unified-filter may skip duplicate budget Lua.
//   - redis: RtbCatalog.RunAuction spends only when authority=rtb; authority=redis leaves budget to unified-filter Lua on /track.
//   - shadow (BudgetAuthorityShadow): RunAuctionEval only inside RtbCatalog; no BudgetStore mutation.
//
// Invariants:
//   - Fail with typed NoBidReason; never panic on corrupt catalog (fault_test.go matrix).
//   - RunAuctionEval must not mutate BudgetStore balances (runAuction spend=false branch).
//   - Catalog snapshot never mutated in place after publish; catalogSlicesValid guards slice length parity.
//   - rankCandidates stops at rankMaxScanCandidates with NoBidScanLimit (core.mdc scan p99 < 500).
//
// Tradeoffs:
//   - SoA catalog vs pointer graph: CampaignAuctionRegistry parallel slices (Bids, CTRPPM, GeoHashes, Weights, ...)
//     plus GeoBucketSoA/TargetBucketSoA candidate indices for cache-friendly linear scan; creativeCacheSoA avoids per-candidate
//     pointer chase. Cold rebuild swaps atomic.Pointer[catalogSnapshot]; hot path loads once per auction with no mutex.
//     Rejected: per-request PG catalog load or mutex-protected map on /track.
//   - 500 scan cap (rankMaxScanCandidates): bounds worst-case auction CPU when buckets are large; returns NoBidScanLimit
//     instead of unbounded catalog walk. Presorted buckets + score floor early break keep typical scans well under cap.
//   - RTB_MODE shadow vs live: shadow runs RunAuctionEval + metrics only (applyRtbAuction returns without rewrite);
//     live runs RunAuction, may rewrite evt.CampaignID and reject on NoBidReason before FilterEngine. Rejected: shadow mode
//     that still debits budget or rewrites campaign (would skew PG/CH funnels during soak).
//   - Budget authority redis vs rtb: RTB_BUDGET_AUTHORITY=rtb spends in CheckAndSpendAll during RunAuction with optional Redis mirror;
//     redis authority leaves financial debit to unified-filter Lua so RTB winner selection and budget enforcement stay in one layer.
//     BudgetAuthorityShadow forces eval-only for catalog glue and reconcile workers. Rejected: double debit (RTB CAS + Lua) without mirror skip.
//
// Forbidden:
//   - encoding/json on production auction or /openrtb/bid response assembly (codec in internal/openrtb + ingest).
//   - Postgres, ClickHouse, or outbox inside RunAuction / rankCandidates loops.
//   - Import internal/controlplane admin or internal/fraud ML scoring.
//
// Verify:
//
//	go list -e ./internal/rtb/
//	go test ./internal/rtb/... -short -count=1
//	go test ./internal/rtb/ -short -run TestAuction_secondPrice_basic -count=1
//	go test ./internal/rtb/ -short -run TestFault_NilRequest -count=1
//	make test-alloc-gate
//	go test ./internal/ingest/ -short -run TestParseOpenRTB26_fields -count=1
package rtb
