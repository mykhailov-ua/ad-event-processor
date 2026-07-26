# Real-Time Bidding

In-process auction on `/track` before `FilterEngine.Check`. Hot path: [HOT_PATH.md](./HOT_PATH.md). Budget authority: [DATA_LAYER.md](./DATA_LAYER.md) Part I. Open gaps: [OPEN_GAPS.md](./OPEN_GAPS.md).

| Metric | Target |
| :--- | :--- |
| `RunAuction` p99 | < 15 us |
| Candidates scanned p99 | < 500 |
| Heap allocations | 0 per auction |

---

## Architecture

RTB is not a standalone OpenRTB exchange endpoint. It runs inside `processTrack()`:

Ingest (`/track`) -> ensureIngestGeo -> applyRtbAuction (`off` skip | `shadow` eval | `live` debit + replace `campaign_id`) -> FilterEngine -> settlement stream.

### Package layout

| Component | Path | Role |
| :--- | :--- | :--- |
| Auction core | `internal/rtb/` | Registry, ranking, clearing, `BudgetStore`, snapshot |
| Tracker glue | `internal/ingestion/rtb_*.go` | Catalog sync, track integration, OpenRTB parse |
| Control plane | `internal/management/handler_rtb.go`, `service_rtb_deals.go`, `service_bid_floor.go` | Deals, lint, floors |
| Configuration | `internal/config/rtb.go`, `env.go` | `RTB_MODE`, authority, targeting index |

### Data flow

| Path | Entry | Work |
| :--- | :--- | :--- |
| Cold | `SyncRtbCatalog`, `ReloadRtbDeals`, `UpdateCampaigns` | Rebuild geo shards, SoA buckets, targeting index, presort |
| Hot | `RunAuction` -> `rankCandidates` | Scan materialized SoA in O(window); CAS budget debit |

`Registry.catalog` is `atomic.Pointer[catalogSnapshot]` swapped on rebuild; readers never take writer locks.

---

## Auction engine

| Capability | Detail |
| :--- | :--- |
| Winner | `bid x CTR` effective score (`auction_ranking.go`); CTR in PPM; zero CTR -> 1.0 |
| Tie-break | Higher `Weight`, then higher raw `Bid` |
| Early exit | `if score < maxScore { break }` after presort (bucket must be fully sorted) |
| Clearing | First- or second-price + reserve (`RTB_CLEARING_MODE=first`) |
| Geo | 64 shards via `req.GeoHash & 63` |
| Targeting | `RTB_TARGETING_INDEX=1` -> inverted index on geo/device/category |
| Budget | CAS in `CheckAndSpendAll`: campaign, customer pool, daily cap with rollback |
| Shadow | `RunAuctionEval` without spend (`RTB_MODE=shadow`) |
| Snapshot | `SaveSnapshot` / `LoadSnapshot` wire v4 at `RTB_SNAPSHOT_PATH` |
| Deals | `DealIndex` atomic swap; floors enforced on hot path |

### Tracker integration

| Item | Detail |
| :--- | :--- |
| Modes | `off`, `shadow`, `live` via `RTB_MODE` |
| OpenRTB 3.0 parse | Substring scan in `openrtb_parse.go`; 0 allocs |
| Deal floors | Max of publisher floor, Postgres deal, Redis `rtb:floor:{id}` |
| Catalog sync | Hybrid metadata -> bid/CTR; periodic registry rebuild |
| Budget authority | `redis`: Lua debit; `rtb`: in-process budget, skip Lua debit |
| Reconcile | Periodic Redis vs RTB divergence sampler |
| Pubsub reload | `rtb:catalog:reload` triggers deals + catalog rebuild |

### Control plane

| Item | Detail |
| :--- | :--- |
| PMP deals | `/admin/rtb/deals` |
| Bid request lint | `POST /admin/rtb/validate-bid-request` (cold path) |
| Shadow diff | `GET /admin/rtb/shadow-diff` |
| Floor optimizer | ClickHouse outcomes -> Redis `rtb:floor:*` |
| Outbox | `RELOAD_RTB_CATALOG` -> Redis pubsub |

### Observability

| Metric | Purpose |
| :--- | :--- |
| `ad_rtb_auction_duration_seconds` | Sampled 1/128 |
| `ad_rtb_auction_candidates_scanned` | Scan cost |
| `ad_rtb_auction_no_bid_total{reason}` | Pre-bound labels |
| `ad_rtb_shadow_winner_mismatch_total` | Shadow divergence |
| `ad_rtb_budget_reconcile_high` | Budget divergence gate |

---

## Environment knobs

| Variable | Default | Description |
| :--- | :--- | :--- |
| `RTB_MODE` | `off` | `shadow` = eval + metrics; `live` replaces `campaign_id` |
| `RTB_BUDGET_AUTHORITY` | `redis` | `rtb` spends in `CheckAndSpend`; Lua skips budget debit |
| `RTB_CLEARING_MODE` | second-price | `first` for first-price |
| `RTB_TARGETING_INDEX` | `true` | Geo + device + category inverted index |
| `RTB_SNAPSHOT_PATH` | - | Budget/catalog snapshot file |
| `RTB_CATALOG_RELOAD_CHANNEL` | `rtb:catalog:reload` | Pubsub reload |
| `RTB_RECONCILE_INTERVAL_MS` | `30000` | Divergence sampler interval |
| `RTB_BUDGET_DIVERGENCE_THRESHOLD_MICRO` | `1000` | Reconcile alert threshold |
| `RTB_RECONCILE_SAMPLE_SIZE` | `32` | Campaigns per tick |
| `RTB_HYBRID_MAX_RPS_PER_NODE` | `5000` | Hybrid balancer metadata |
| `RTB_PREBID_IVT` | `false` | Pre-bid datacenter/proxy gate (R17) |

System setting `rtb_budget_authority` in `system_settings` can override via `RtbAuthorityController`.

---

## Hot-path engineering

RTB is the reference SoA auction loop. Policy: [HOT_PATH.md](./HOT_PATH.md).

### Structure of arrays

Shard registry holds parallel slices (`Bids`, `DeviceMasks`, ...). Bucket `candidateBucketSoA` duplicates hot fields in iteration order. Cold path materializes bucket rows; hot path indexes `bids[pos]`, not `reg.Bids[catalogIdx[pos]]`.

### BCE in `rankCandidates`

```go
if bucketStart < 0 || bucketEnd < bucketStart || bucketEnd > soa.len() {
    return ..., NoBidCorruptCatalog
}
catalogIdx := soa.CatalogIdx[bucketStart:bucketEnd]
bids := soa.Bids[bucketStart:bucketEnd]
for pos := 0; pos < len(catalogIdx); pos++ {
    bid := bids[pos]
}
```

Verify: `go tool objdump` - no `CALL runtime.panicIndex` in loop body.

### False-sharing padding

```go
type AlignedBudget struct {
    Value int64
    _     [7]int64  // pad to 64 bytes
}
```

### Budget authority modes

| Mode | Debit location | Lua budget |
| :--- | :--- | :--- |
| `redis` | `unified-filter.lua` / `budget-fast.lua` | Active |
| `rtb` | `internal/rtb/budget_spend.go` CAS | Skipped |

Mirror Redis into `BudgetStore` when authority is `rtb`. Reconcile sampler compares divergence.

### Verification

```bash
go test -bench=BenchmarkAuction -benchmem ./internal/rtb/
go test -gcflags="-m" ./internal/rtb/ 2>&1 | rg 'does not escape'
go test -run TestChaos ./internal/rtb/...
```

| Criterion | Requirement |
| :--- | :--- |
| Allocations | 0 B/op on `BenchmarkAuction` |
| BCE | No `panicIndex` in `rankCandidates` loop |
| Atomics | No `atomic.Load` inside candidate loop |
| Interfaces | No `interface{}` / closures in auction path |
| Strings | No `+` concat per request in parse/rank |

---

## File reference

| File | Role |
| :--- | :--- |
| `internal/rtb/auction.go` | `RunAuction`, `RunAuctionEval`, clearing |
| `internal/rtb/auction_rank.go` | `rankCandidates`, BCE window |
| `internal/rtb/auction_ranking.go` | `effectiveScore`, CTR PPM |
| `internal/rtb/auction_clearing.go` | First/second price, reserve |
| `internal/rtb/catalog_registry.go` | Shard rebuild, atomic publish |
| `internal/rtb/catalog_bucket_soa.go` | Bucket materialization |
| `internal/rtb/catalog_bucket_sort.go` | Presort |
| `internal/rtb/catalog_geo_index.go` | Geo bucket build |
| `internal/rtb/catalog_targeting_index.go` | Inverted index |
| `internal/rtb/budget_store.go` | Aligned slots |
| `internal/rtb/budget_spend.go` | CAS debit, rollback |
| `internal/rtb/metrics.go` | Sampled metrics |
| `internal/rtb/persistence.go` | Snapshot v4 |
| `internal/rtb/deal_index.go` | PMP deal snapshot |
| `internal/ingestion/rtb_track.go` | Track integration |
| `internal/ingestion/rtb_catalog.go` | `RtbCatalog` facade |
| `internal/ingestion/rtb_sync.go` | Catalog sync |
| `internal/ingestion/openrtb_parse.go` | Hot OpenRTB 3.0 parse |
| `internal/ingestion/openrtb_validate.go` | Cold 2.6/3.0 lint |

Not on hot path today: `HybridBalancer.SelectAndShard` (built, uncalled). Open RTB exchange surface and live production gates: [OPEN_GAPS.md](./OPEN_GAPS.md).
