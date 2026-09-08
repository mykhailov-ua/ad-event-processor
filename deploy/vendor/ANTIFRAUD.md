# Antifraud reference

Operator and engineering reference for tracker fraud layers, cold-path IVT/ML workers, and enforcement side effects. Canonical code: `internal/filter`, `internal/ingest`, `internal/fraud`, `internal/track`, `internal/outbox`, `cmd/ivt-detector`, `cmd/fraud-scorer`, `cmd/tracker/wire.go`. Architecture context: [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md).

This document describes **what the tree does today**. It does not claim features that exist only as registry entries, admin flags without hot-path readers, or marketing SLAs from unit microbenches.

---

## Hot vs cold

| Path | Binaries / packages | Allowed synchronous I/O on request thread |
| :--- | :--- | :--- |
| Hot | `cmd/tracker`, `internal/ingest`, `internal/filter`, `internal/stream` producers | Go-local filters; at most **one** Redis `EVALSHA` per accepted event (zero when local-quanta full-skip eligible). Optional one `INCR` for ingress RPD in `EntitlementsFilter`. |
| Cold | `cmd/control`, `cmd/processor`, `cmd/ivt-detector`, `cmd/fraud-scorer`, `internal/fraud`, `internal/outbox` | Postgres, ClickHouse, batch ML, management HTTP enqueue, outbox apply. |

Hard boundaries:

- Tracker **must not** import `internal/fraud` scoring (`boundaries.mdc`).
- `/openrtb/bid` runs in-process auction only; it does **not** run the full `FilterEngine` chain.
- ML inference (LightGBM) runs only in `cmd/fraud-scorer` / embedded scorer inside `cmd/ivt-detector` when not standalone. Tracker reads `ml:score:boost:{campaign_id}` from an in-memory snapshot (`SettingsWatcher`), not from per-request model calls.

Ingress SLA ceilings (`core.mdc`): handler p95 < 50 ms, p99 < 80 ms; unified-filter Lua p99 < 10 ms per shard in load tests / Prometheus. Isolated Go microbenches (for example `BenchmarkFilterFraudBoost`) are **not** production `/track` SLA proof.

---

## End-to-end hot path

```
HTTP /track|/click -> parse -> TryReserve (stream/broker admission)
  -> FilterEngine.Check (sync on PinnedWorkerPool worker)
       -> local filters accumulate fraud signals (no Redis between cheap gates)
       -> layer decision (L1 reject | L2 shadow | pass)
       -> UnifiedFilter last (Redis Lua debit unless shadow / short-circuit / local-quanta full-skip)
  -> publish main stream (unless L1 reject)
  -> HTTP 202/302 (accept or silent fraud) | 403 (hard fraud) | 503 (infra)
```

Fraud **L1 reject** also enqueues a separate Redis fraud stream (`FraudStreamWriter`) for analytics. L2 shadow events go to the **main** ingest stream with `ShadowEvent=true` and **no** budget debit.

---

## Filter chain order (production)

Registered in `cmd/tracker/wire.go` (cheapest local gates first; **UnifiedFilter last**):

| Order | Filter | Fraud role |
| ---: | :--- | :--- |
| 1 | `LicenseFilter` | License gate; no fraud signals |
| 2 | `LicenseRPSFilter` | License RPS; no fraud signals |
| 3 | `EmergencyBreakerFilter` | Ops breaker; no fraud signals |
| 4 | `GeoFilter` | Geo targeting; optional `accept_lang_geo_mismatch` |
| 5 | `ScheduleFilter` | Schedule gate; no fraud signals |
| 6 | `VPPFilter` | Volume protection; no fraud signals |
| 7 | `FraudFilter` | `datacenter_ip` (Geo anonymous + DC ASN table) |
| 8 | `ResidentialProxyFilter` | `residential_proxy` (intel table or ring) |
| 9 | `TCPMSSFilter` | `tcp_mss_anomaly`, `tcp_tunnel_mss` |
| 10 | `DeviceFilter` | TLS blocklist, device/JA4/OS/TCP SYN signals |
| 11 | `L7WireFilter` | Sec-Fetch, Client Hints, TLS ALPN, H2, header order |
| 12 | `JSONSerializationFilter` | `json_serialization_bot` |
| 13 | `BehaviorTelemetryFilter` | `behavior_telemetry_missing`, `behavior_bezier_bot` |
| 14 | `ConsentFilter` | Consent gate; no fraud signals |
| 15 | `SegmentFilter` | Segment gate; no fraud signals |
| 16 | `EntitlementsFilter` | Ingress RPD (`INCR`); CGNAT bypass for velocity only |
| 17 | `UnifiedFilter` | Placement blacklist, fraud blacklist, TTC, Lua budget |

Inside `UnifiedFilter.checkPass`: placement `HEXISTS`, fraud blacklist `SISMEMBER`, optional Go TTC signals, then Lua debit (or local-quanta path).

Click/landing hooks (`internal/ingest/landing_bundle.go`) may add signals **before** `FilterEngine.Check`: `attestation_missing`, `ipv4_rotation`, `datacenter_ip` (IPv6 rotation shadow).

---

## Fraud accumulator

Implementation: `internal/filter/engine.go`, `internal/filter/util.go` (`fraudReasonRegistry`).

| Property | Value |
| :--- | :--- |
| Max distinct signals per event | 4 (`maxFraudSignals`) |
| Score | Sum of signal weights, capped at 100 |
| Dedup | Same `FraudReasonID` not added twice |
| ML boost | Added once per check from snapshot: `score = min(100, score + boost)` |
| Stored on event | `FraudScore`, comma-separated `FraudReason` codes, `LayerDesyncCount` |

### Signal registry (hot path)

| Code | Weight | Flag | Typical source |
| :--- | ---: | :--- | :--- |
| `l3_blocklist` | 100 | L3 | `FraudBlacklistFilter` / unified pre-Lua (`blacklist:fraud`) |
| `datacenter_ip` | 45 | L1-high | `FraudFilter` (anonymous GeoIP or DC ASN table) |
| `low_ttc` | 45 | L1-high | Unified Lua / Go TTC precheck |
| `tls_blocklist` | 45 | L1-high | `DeviceFilter` (dynamic TLS hash blocklist) |
| `moderator_ip` | 45 | L1-high | **Registered only; no hot-path `addFraudSignal` call in tree** |
| `missing_imp_ts` | 35 | L2-weak | Unified Lua / Go TTC |
| `device_mismatch` | 35 | L2-weak | `DeviceFilter` |
| `tcp_mss_anomaly` | 35 | L2-weak | `TCPMSSFilter` |
| `tcp_tunnel_mss` | 35 | L2-weak | `TCPMSSFilter` |
| `tcp_syn_os_mismatch` | 35 | L2-weak | `DeviceFilter` |
| `json_serialization_bot` | 35 | L2-weak | `JSONSerializationFilter` |
| `os_fingerprint_mismatch` | 35 | L2-weak | `DeviceFilter` (TTL/window vs UA) |
| `ipv4_rotation` | 35 | L2-weak | Landing IPv4 rotation hook |
| `residential_proxy` | 35 | L2-weak | `ResidentialProxyFilter` |
| `attestation_missing` | 35 | L2-weak | Landing light attestation hook |
| `sec_fetch_anomaly` | 35 | L2-weak | `L7WireFilter` |
| `client_hints_mismatch` | 35 | L2-weak | `L7WireFilter` |
| `tls_alpn_mismatch` | 35 | L2-weak | `L7WireFilter` |
| `h2_settings_mismatch` | 35 | L2-weak | `L7WireFilter` |
| `h2_pseudo_order_mismatch` | 35 | L2-weak | `L7WireFilter` |
| `h2_downgrade_artifact` | 35 | L2-weak | `L7WireFilter` |
| `header_order_mismatch` | 35 | L2-weak | `L7WireFilter` |
| `accept_encoding_mismatch` | 35 | L2-weak | `L7WireFilter` |
| `accept_lang_geo_mismatch` | 35 | L2-weak | `GeoFilter` when campaign flag enabled |
| `tls_ja4_mismatch` | 35 | L2-weak | `DeviceFilter` (browser corpus) |
| `behavior_telemetry_missing` | 35 | L2-weak | `BehaviorTelemetryFilter` |
| `behavior_bezier_bot` | 35 | L2-weak | `BehaviorTelemetryFilter` |

Flag groups: `FraudSignalL3`, `FraudSignalL1High`, `FraudSignalL2Weak` in `internal/filter/util.go`.

### Layer desync (analytics)

`LayerDesyncCount` counts how many **distinct wire layers** fired among: TCP/SYN-OS, TLS JA4, Client Hints, Sec-Fetch, HTTP/2 (`internal/filter/filters_chain.go`). Used in fraud reports; does not change layer decision by itself.

---

## Tier mapping (campaign thresholds)

`MapFraudTier` in `internal/filter/engine.go` maps accumulated score to tier using per-campaign thresholds (Postgres `campaigns.fraud_threshold_*`), with defaults:

| Field | Default |
| :--- | ---: |
| `fraud_threshold_pass` | 30 |
| `fraud_threshold_suspect` | 60 |
| `fraud_threshold_ivt` | 80 |
| `fraud_threshold_block` | 100 |

Presets (`domain.ResolveFraudPreset`): `conservative` (40/70/90/100), `balanced` (defaults), `aggressive` and `enhanced_defense` (20/45/65/85), `social_in_app` (defaults). Legacy preset alias: `gray_market` -> `enhanced_defense`.

Tier influences **L2 vs pass** when only weak signals exist; it does not override L3 or two L1-high signals (see below).

---

## Layer decision (L1 / L2 / none)

`decideFraudLayer` in `internal/filter/filters_chain.go`:

| Condition | Layer | UnifiedFilter Lua debit | Main stream | Typical HTTP on L1 |
| :--- | :--- | :--- | :--- | :--- |
| No signals | None | Runs | Yes, normal | n/a |
| L3 signal (`l3_blocklist`) | L1 reject | Skipped (short-circuit before debit filter) | No | 403 or silent accept |
| >= 2 L1-high signals | L1 reject | Skipped | No | 403 or silent accept |
| >= 1 L1-high, or >= 1 L2-weak, or tier suspect/IVT/block | L2 shadow | Skipped (`SetSkipBudgetDebit`) | Yes, `ShadowEvent=true` | Normal accept |
| Otherwise | None | Runs | Yes, normal | n/a |

**Short-circuit before UnifiedFilter:** when `shouldShortCircuitFraudBudget()` (L3 or two L1-high), `FilterEngine` applies the layer decision and **breaks** before reaching `UnifiedFilter` (`internal/filter/engine.go`).

**L2 shadow budget invariant:** integration holdout `TestFilterEngine_shadowSkipsUnifiedBudgetDebit_holdout` proves Redis campaign spend unchanged after shadow accept.

### Silent reject vs hard reject (L1 only)

Campaign flag: `silent_reject_enabled` (legacy JSON alias on PATCH: `ghost_ivt_enabled`). Read on hot path via registry snapshot (`internal/track/processor.go` `FraudOutcome`).

| `silent_reject_enabled` | HTTP | `SilentRejectEvent` on event | Fraud stream |
| :--- | :--- | :--- | :--- |
| true | 202 (`/track`) or 302 (`/click`) | true | Enqueued |
| false | 403 | false | Enqueued |

ClickHouse column for analytics: `silent_reject_event` (not legacy `ghost_event` in new code).

**Important:** ML `silent_reject` **enforcement** adds the client IP to `blacklist:fraud` via outbox. It does **not** `UPDATE campaigns SET silent_reject_enabled`. Per-IP decoy HTTP still requires the campaign flag on the hot path (`TestFault_FraudSilentRejectAddsBlacklistNotCampaignFlag`).

---

## Redis, caches, and honest I/O

| Check | Steady-state behavior | Fail mode |
| :--- | :--- | :--- |
| Fraud blacklist `blacklist:fraud` | In-process positive-only cache (5 s TTL); `SISMEMBER` on miss | Redis error -> **503** fail-closed (`FraudBlacklistFilter`) |
| Placement blacklist | Per-campaign hash; in-process cache; `HEXISTS` on miss | Redis error propagates |
| Ingress RPD | `EntitlementsFilter`: one `INCR` per event per customer/day key | Redis error -> reject |
| Lua unified filter | One `EVALSHA` when not local-quanta full-skip / not L2 shadow short-circuit | Breaker open -> 503 |
| ML boost | Go `SettingsWatcher` snapshot only; Redis updated async via outbox or `fraud-scorer` microbatch | Missing key -> boost 0 |

`UnifiedFilter.SetIngressRPDHandledExternally(true)` in tracker wire tells Lua to skip duplicate ingress-RPD work when `EntitlementsFilter` already incremented.

`LOCAL_QUOTA_MODE=live` can skip sync `EVALSHA` for eligible traffic; coordinate with `StreamProducer` / `fcap:ignored` so there is a single stream writer (`hot-path.mdc`).

---

## ML score boost (hot read, cold write)

| Stage | Behavior |
| :--- | :--- |
| Write | `cmd/fraud-scorer` microbatch or outbox `ML_SCORE_BOOST` -> Redis `ml:score:boost:{campaign_id}` (TTL default **900 s**, `scorer.ScoreBoostTTL`) |
| Read | `FilterEngine` adds campaign boost to fraud accumulator before tier/layer decision |
| Pause | Microbatch pauses scoring when processor stream lag exceeds `MaxStreamLagSec` (default 30 s); may refresh TTL only |

License: `ml_fraud_boost` entitlement required for `cmd/fraud-scorer`. Suspect-tier batch scores map to `boost` action in `fraud_scoring_rule.go`.

---

## CGNAT / mobile carrier policy

When **both** campaign `cgnat_ip_policy_enabled` and global `CGNAT_MOBILE_IP_BYPASS` (or global bypass alone) apply, and IP ASN is a configured mobile carrier:

- Skip **IP-frequency** signals only: ingress RPD (`EntitlementsFilter`), unified TTC bypass paths tagged `ingress_rpd` / CGNAT helpers (`internal/filter/cgnat_policy.go`).
- Does **not** bypass: datacenter ASN, TLS blocklist, L3 blacklist, attestation, budget Lua, geo targeting.

---

## CDN and edge header caveat

TCP MSS, TTL, SYN signature, TLS JA3/JA4, and HTTP/2 wire signals depend on edge-injected headers (`X-TCP-MSS`, `X-TLS-JA3`, etc.) or direct client TCP visibility. Behind CDN/ALB termination many signals are absent or stale: filters **fail open** (no signal) when headers missing (`DeviceFilter` increments `OSFingerprintSkippedTotal` for `no_tcp_headers`). Disable or expect degraded signal rate; do not treat edge TCP fingerprinting as residential-fraud proof (`edge.mdc`).

---

## Cold path: `cmd/ivt-detector`

Requires ClickHouse, `IVT_DETECTOR_ENABLED=true`, license module `ivt_ml_detector`, compose profile `analytics-ml` unless otherwise deployed.

Loop: `fraud.Detector.Run` -> CH rules -> management HTTP -> Postgres outbox -> `OutboxWorker` -> Redis/PG side effects.

**Backpressure:** when outbox `PENDING` (excluding enforcement event types in `OutboxEnforcementEventTypes`) exceeds `IVT_DETECTOR_OUTBOX_PENDING_LIMIT` (default **500**), detector returns `ErrOutboxBackpressure` and skips the scan (`internal/fraud/outbox_backpressure.go`).

### IVT rules (ClickHouse batch)

Registered in `fraud.NewAnalyzerRegistry` (`internal/fraud/rules.go`):

| Rule name | CH source | Default enforcement |
| :--- | :--- | :--- |
| `high_click_to_imp_ratio` | clicks/impressions window | `BlockIP` management API (legacy path) |
| `shared_fingerprint_cluster` | many IPs per UA hash | `BlockIP` |
| `campaign_ctr_spike` | per-campaign CTR | `BlockIP` |
| `interval_bot` | low variance inter-click intervals | `BlockIP` |
| `rtt_split_tunnel` | `rtt_split_delta_ms` on clicks | `silent_reject` enqueue (TTL 3600 s) when enabled |
| `mobile_biometrics` | flat gyro / motionless mobile conversions | `silent_reject` enqueue |
| `tcp_edge_correlation` | edge Redis fingerprints vs CH TLS | `silent_reject` enqueue when JA3 suspicious |
| `datacenter_asn` | CH IPs + ASN classifier | `BlockIP` when ASN classifier configured |
| `fraud_scoring_shadow` | `ml_features_1m` + LGBM | see ML table below |

Rules without explicit `Action` use detector `default` branch: synchronous `management.BlockIP` (not `EnqueueFraudThreatBatch`). ML and some correlation rules set `Action` explicitly.

Configurable via `AnalyzerConfig` / env (window, min clicks, RTT split thresholds, mobile biometrics thresholds). See `fraud.DefaultAnalyzerConfig()`.

Optional: `fraud.ResidentialIntelEnricher` goroutine when external feed + SKU allow (cold enrichment, not hot-path filter).

Embedded LGBM in ivt-detector is **skipped** when `FRAUD_SCORER_STANDALONE=1` (use `cmd/fraud-scorer` instead).

---

## Cold path: `cmd/fraud-scorer`

Batch LightGBM sidecar: `FRAUD_SCORING_ENABLED=true`, `ml_fraud_boost` license, model path from config. Registers `fraud_scoring_rule`; enforces via `EnqueueFraudThreatBatch` only (no direct Redis from scorer process).

Policy: `fraud.ResolvePolicyConfig` / `DecideWithCampaign` with structural and residential-proxy adjustments (`internal/fraud/scoring_policy.go`). Shadow scores may insert to ClickHouse when write conn configured.

### ML batch actions (`fraud_scoring_rule`)

Uses per-campaign thresholds and `silent_reject_enabled` **only to choose ML enforcement action**, not to flip the flag:

| ML tier | Action | Side effect |
| :--- | :--- | :--- |
| Suspect | `boost` | Outbox `ML_SCORE_BOOST` -> Redis boost key |
| IVT | `blacklist` or `silent_reject` | `silent_reject` when campaign `silent_reject_enabled`; else `blacklist` |
| Block | `blacklist` | `ML_BLACKLIST_ADD` / blacklist ops |
| Pass | none | Debug log only; shadow CH insert may still run |

Wire alias: enqueue accepts `ghost` as `silent_reject` (`internal/fraud/detector_run.go`).

`silent_reject` ML action TTL default **300 s** on blacklist entry in scoring rule; boost TTL **900 s**.

---

## Outbox enforcement (async truth)

Management HTTP (`internal/fraud/admin_hooks`) inserts outbox rows; `internal/outbox` worker applies:

| Event type | Effect |
| :--- | :--- |
| `ML_SCORE_BOOST` | `SET ml:score:boost:{campaign_id}` on all Redis shards; pub/sub campaign reload |
| `ML_SILENT_REJECT` / `ML_GHOST_IVT` | PG blacklist row + `SADD blacklist:fraud` + invalidation pub/sub |
| `ML_BLACKLIST_ADD` | Same fast lane as ML blacklist |
| `UPDATE_BLACKLIST` | Operator blacklist reasons (`blacklist:{reason}`) |

**Invariant:** hot handler never writes `blacklist:fraud` directly; cold outbox is authoritative for ML/IVT enforcement.

---

## Conversion smart reject (processor / postback)

Cold-path `ConversionRejectApplier` (`internal/postback/conversion_reject.go`) runs on conversion events **before payout**, not on `/track`.

| Reason code | Meaning |
| :--- | :--- |
| `conversion_no_click` | No matching click row |
| `conversion_low_ttc` | TTC below threshold |
| `conversion_duplicate` | Duplicate goal |
| `conversion_ip_drift` | IP differs from click |
| `conversion_datacenter_ip` | Datacenter classification on conversion IP |

When campaign `silent_reject_enabled`, rejected conversions set `SilentRejectEvent` (same analytics flag as hot-path silent accept). Rejected conversions skip postback outbox enqueue (`TestConversionReject_rejectSkipsOutboxEnqueue`).

Global and per-campaign rules merge via `mergeConversionRejectConfig`. Click store unavailable: fail-open to `pending_validation` (degraded), not silent approve.

---

## Edge and XDP (enterprise)

| Component | Role | Fraud scope |
| :--- | :--- | :--- |
| nginx Lua (`deploy/nginx/lua/`) | Rate limit, slot map, access checks, proxy to tracker | May 403 before tracker; does not run unified budget Lua |
| `cmd/edge-bpf-sync` | Redis -> BPF maps | Syncs block lists and fingerprints |
| `cmd/edge-xdp` (`edge_filter.c`) | L3/L4 drop at NIC | Listed hosts, flood control, syn-cookie pressure |

XDP drops **known L3/L4 endpoints and floods**, not application-layer residential fraud. Rotating proxies and CDN clients evade host-based maps. ICMP PMTUD Type 3 Code 4 is passed for tracker path (not global ICMP drop). Token bucket uses sub-interval credit (`TestTokenBucket_subintervalCredit_holdout`).

---

## Admin and reporting surfaces

| Surface | Path / package |
| :--- | :--- |
| Campaign fraud PATCH | `/api/v1/campaigns/{id}/fraud` |
| Fraud admin | `/api/v1/fraud/*` (`internal/fraudadmin`) |
| Ops ML / threats | `/api/v1/ops/ml-model`, `/api/v1/ops/fraud-threat`, blacklist ops |
| Reports | `internal/reports/fraud/` (silent-reject funnel, layer desync, signal breakdown, evidence pack; SKU `fraud_dispute_evidence` on Scale+) |
| Blacklist janitor | `fraudadmin.BlacklistJanitor` expires PG blacklist rows |

---


## Ingest sink integrity

Cross-cutting contracts for accept responses, budget reconciliation, broker cutover, and local-quanta full-skip. HTTP status codes name the wire response; **publish** means synchronous `TryReserve` + `publishAcceptedOrRollback` (`ProcessReserved` / `EnqueueReserved` on stream or broker ring). Redis `XADD` / broker WAL flush is async after the response returns.

### Accept response vs reserve + publish

| Entrypoint | Handler | Accept status | Normal accept: publish before response? | Notes |
| :--- | :--- | :--- | :---: | :--- |
| POST `/track` (JSON, protobuf, OpenRTB3-in-JSON) | gnet `deliverGnetTrack`, HTTP/1 offload, `NewHTTPTrackMux` | **202** | **Yes** | `publishAcceptedOrRollback` then `writeGnetTrackAccepted`; publish fail -> **503** + `RollbackDebit` |
| POST `/track` silent-reject fraud | same | **202** | **No** (by design) | `silent_reject_enabled`: fraud stream only; main ingest sink skipped; lease released |
| GET `/click` / landing | `reactClickRedirect` | **302** | **Yes** (when `filterEngine` set) | Smoke/safe-page branches skip publish by design |
| GET `/tg/click` | `reactTelegramClick` | **302** | **Yes** on accept | Fraud decoy: **302** without main-stream publish |
| GET `/tg/impression` | `reactTelegramImpression` | **204** | **Yes** on accept | Same fraud exception as click |
| POST `/tg/bid` | `reactTelegramBid` | **200** | N/A | RTB auction only; no stream admission |
| POST `/openrtb/bid` | `reactOpenRTBBid` | **200** / **204** | N/A | In-process auction; no `TryReserve` |
| `filterEngine == nil` on POST `/track` | handler fallback | **202** | **Yes** | No Lua debit; still `TryReserve` + publish before 202 |

OpenRTB impressions embedded in POST `/track` JSON follow the `/track` rows, not `/openrtb/bid`.

**Verify:**

```bash
bash scripts/ci/static/budget_rollback_gate.sh
go test ./internal/ingest/ -short -run 'PublishAcceptedOrRollback|RollbackDebit' -count=1
```

### Budget reconciliation (cold path)

| Component | Role |
| :--- | :--- |
| `GlobalSpendReconciler` (`internal/reconciliation/global_spend.go`) | Batches regional spend deltas into PG + Redis under `sync_idempotency` |
| `ReconWorker` (`internal/reconciliation/worker.go`) | Dirty-set snapshot: `budget:dirty_campaigns`, per-campaign sync keys from unified-filter Lua |
| `AssertBudgetInvariant` (`internal/domain/budget`) | PG `current_spend <= budget_limit` (+/-1 micro-unit) on spend/fault tests |
| Dirty sync sets in Lua | `SADD budget:dirty_campaigns`, campaign/customer sync `INCRBY`; rollback Lua reverses on post-debit failure |

**Operator alerts** (`deploy/monitoring/prometheus.rules.yaml`):

| Alert | Signal | Meaning |
| :--- | :--- | :--- |
| Post-debit enqueue failure | `rate(ad_stream_producer_post_debit_rejected_total[2m]) > 0` | Debit ran, sync enqueue failed; rollback path active |
| Reconciliation job failure | `rate(ad_reconciliation_runs_total{status="failed"}[5m]) > 0` | Cold recon worker error |
| Financial discrepancies | `rate(ad_reconciliation_discrepancies_total[5m]) > 5` | Redis/PG drift detected |
| Budget drift | `max(ad_reconciliation_drift_micro) > 50000000` | Snapshot drift above 50M micro-units |
| Budget cache miss to PG | `rate(ad_budget_cache_miss_pg_total[5m]) > 0` | Hot path fell back to `GetByID` for budget key |
| Lua budget key miss | miss rate vs HTTP rate | Registry sync lag or evicted keys |

CH lag without spend growth: compare `clickhouse` ingest lag metrics with `ad_events_processed` / reconciliation drift; no single alert encodes both — use dashboard join.

### Broker-primary and shadow cutover

| Phase | Config | Contract |
| :--- | :--- | :--- |
| Shadow | `BROKER_SHADOW_MODE=1`, `CH_INGEST_SOURCE` still stream or dual | Broker producer writes WAL; consumer compares to stream path; `ad_broker_ingest_divergence_high` set when threshold breached |
| Cutover | Flip `CH_INGEST_SOURCE=broker` after shadow quiet | Require `ad_broker_ingest_divergence_high == 0` sustained before live cutover |
| WAL semantics | `pkg/broker` mmap ring | **At-least-once on sink side** (CH/broker consumer); does **not** pair debit with rollback — post-debit publish fail still uses `budget-rollback.lua` or local-quanta refund |

**Verify:** `make test-fault` (`TestFault_BrokerShadowCutover_NoEventLoss`, `TestFault_BrokerLiveConsumer_CorruptPayload`); integration tier for real broker wiring.

### Local quanta full-skip (`LOCAL_QUOTA_MODE=live`)

| Stage | Behavior |
| :--- | :--- |
| Eligibility | `LOCAL_QUOTA_MODE=live`, quota enabled, fast path, no full Lua path; `LocalQuantaFullSkipEligible` |
| Debit | `TrySpendDebit` on `LocalQuantaLedger` (~16 ns); skips sync `EVALSHA` when eligible |
| Async stream | `LocalQuantaStreamPublisher.Enqueue` (MPSC ring per shard); worker runs `XADD` / refill via `local-quota-refill.lua` |
| Click idempotency | `LocalClickIdemCache.TryClaim` per `click_id` before enqueue |
| Post-debit publish fail (ingest) | `RollbackDebit(isLocalQuanta=true)` -> `RefundDebit` + `PublishReturn`; idempotency guard `localRollbackGuard` (same TTL as click idempotency) |
| Redis path rollback | `budget-rollback.lua` with `{campaign_id}rollback:click:{click_id}` SET NX guard |

**Risk class:** local debit is synchronous on Tier B; async `LocalQuantaStreamPublisher` XADD runs only after `publishAcceptedOrRollback` succeeds (`evt.LocalQuantaDebitMicro` pending until `FinalizeLocalQuantaPublish`). Main publish fail -> **503** + `RollbackDebit` + `ad_local_quota_rollback_total` (never 202 with uncertain debit).

Partition / duplicate rollback: Redis path uses Lua guard (code `2`); local-quanta path uses `localRollbackGuard` + pending-debit rollback. Operator: fail-closed **503** over accept-and-hope; do not weaken rollback for 202.

**Monitor:** `ad_local_quota_full_skip_*`, `ad_local_quota_stream_drop_total`, `ad_local_quota_stream_write_error_total`, `ad_local_quota_rollback_total`, `ad_local_quota_finalize_failed_total` (alerts in `prometheus.rules.yaml`). Holdout: `TestUnifiedFilter_RollbackDebit_LocalQuanta`, `TestLocalQuantaPendingDebit_publishFail_holdout` (do not disable).

**Verify:**

```bash
bash scripts/ci/static/budget_rollback_gate.sh
go test ./internal/ingest/ -short -run 'LocalQuanta|RollbackDebit' -count=1
go test ./internal/filter/unified/ -short -run TestBudgetRollback_ -count=1
```

---

## Invariants (summary)

| ID | Invariant | Verify |
| :--- | :--- | :--- |
| H1 | Hot path does not import `internal/fraud` ML | `TestTrackerDepGraphExcludesFraudScoringRuntime` |
| H2 | At most one sync `EVALSHA` per accept (or zero full-skip) | `architecture.mdc`, integration ingest tests |
| H3 | L2 shadow does not debit Redis budget | `TestFilterEngine_shadowSkipsUnifiedBudgetDebit_holdout` |
| H4 | L1 fraud enqueues fraud stream (silent and hard) | `handler_track_sla_fault_test`, `track_bundle.go` |
| H5 | `silent_reject_enabled` controls HTTP decoy only on hot path | `TestFraudReject_holdoutSilentRejectFlag` |
| H6 | ML `silent_reject` adds IP to blacklist, not campaign flag | `TestFault_FraudSilentRejectAddsBlacklistNotCampaignFlag` |
| H7 | Fraud blacklist cache is positive-only (fresh SADD visible) | `TestFraudBlacklistFilter_freshBlacklistNotHiddenByNegativeCache_holdout` |
| H8 | Fraud blacklist Redis error fails closed | `TestFraudBlacklistFilter_redisError_failClosed_holdout` |
| H9 | Stream admission before Lua debit; accept response after publish (normal path) | `TestStreamProducerAdmissionRaceWithoutReserve`; `TestPublishAcceptedOrRollback_holdout` |
| H10 | Post-debit enqueue fail rolls back debit (idempotent) | `budget_rollback_gate.sh`; `TestUnifiedFilter_RollbackDebit_LocalQuanta`; `TestBudgetRollback_idempotent_holdout` |
| H11 | IVT detector pauses on outbox backlog | `ivt_detector_outbox_backpressure` fault proof |
| H12 | Budget PG invariant on spend paths | `AssertBudgetInvariant` |
| H13 | `moderator_ip` signal not emitted by production filters | grep: no `addFraudSignal(...ModeratorIP)` outside registry |

---

## Verification commands

```bash
# Hot-path fraud behavior (unit; not Redis integration)
go test ./internal/ingest/ -short -run 'Fraud|SafePage|TLS|SilentReject|Shadow' -count=1
go test ./internal/filter/ -short -run 'Fraud|MapFraudTier' -count=1

# Integration tier (Redis Lua, shadow debit holdout)
make test-integration

# Cold fraud package
go test ./internal/fraud/ -short -count=1

# Edge token bucket / XDP source holdouts
go test ./internal/edge/ -short -run 'TestTokenBucket_|TestEdgeFilter' -count=1

# Doc vs code parity (no false "eliminated Redis" claims)
bash scripts/ci/naming/antifraud_doc.sh
```

For broker/CH sink wiring claims, use `make test-fault` / integration tier (`anti-slop.mdc`); unit mocks are not production proof.

---

## Open implementation notes

Documented gaps (do not close in docs alone):

- `moderator_ip` exists in `fraudReasonRegistry` and campaign JSON `moderator_intel_enabled`, but **no** production filter calls `addFraudSignal(FraudReasonModeratorIP)`. Moderator intel table loads in `internal/filter/netintel` for future use.
- Per-IP silent HTTP decoy requires campaign `silent_reject_enabled`; ML cannot replace that flag for hot-path response mapping.
- IVT `BlockIP` default path and `EnqueueFraudThreatBatch` path coexist; operators should trace outbox event types per rule when auditing enforcement.
