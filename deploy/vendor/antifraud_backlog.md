# Antifraud ROI backlog (internal)

Shippable antifraud work derived from [ANTIFRAUD_MARKET_ANALYSIS.md](./ANTIFRAUD_MARKET_ANALYSIS.md). Targets payout protection, closed-loop zone cuts, antidetect signal depth, and CGNAT false-positive reduction.

**Not in scope:** PPC Google/Meta IP exclusion sync, signed legal refund PDFs, full mobile biometrics (gyro/touch radius), marketing accuracy claims without holdout tests.

**Canonical implementation truth:** [ANTIFRAUD.md](./ANTIFRAUD.md). Competitive parity items live in [competitive_backlog.md](./competitive_backlog.md).

Cross-reference slugs by name in PRs and docs. Do not close a slug until every applicable gate below is checked.

---

## Priority legend

| Label | Meaning |
| :--- | :--- |
| `roi_p0` | Direct payout or placement waste reduction; ship first |
| `roi_p1` | Strong differentiation; depends on safe-page or CH rollups |
| `roi_p2` | False-positive reduction; tune carefully to avoid mobile-proxy bypass |

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [ ] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [ ] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`, `hot-path.mdc`)
- [ ] No new sync Postgres / ClickHouse / external HTTP on `/track` or filter `Check` (`architecture.mdc`)
- [ ] At most one sync Redis `EVALSHA` per accepted event; no extra round-trips between local filters (`hot-path.mdc`, `architecture.mdc`)
- [ ] Fraud blacklist / placement cache: no new per-event `SISMEMBER` / `HEXISTS` without TTL shard cache (`ANTIFRAUD.md`, `anti-slop.mdc` lie modes)
- [ ] Verification commands pasted in PR with package path (no unrun claims - `quality.mdc`)
- [ ] Holdout or fault test added when behavior is non-obvious (`testing.mdc`)
- [ ] Doc claims match code; no microbench cited as prod SLA (`anti-slop.mdc`)
- [ ] `bash scripts/ci/pr_fast.sh` scoped to touched packages (`ci.mdc`)
- [ ] `bash scripts/ci/antifraud_doc_gate.sh` when touching `deploy/vendor/ANTIFRAUD.md` or this file
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched vendor docs (`naming.mdc`)

Rule: `core.mdc` commit policy (when landing code)

- [ ] Imperative commit title names concrete surface (route, worker, filter, migration)
- [ ] Docs-only antifraud claims ship in the same commit as code (`core.mdc`)

---

## Hot / cold / edge boundary (read before coding)

| Surface | Allowed | Forbidden |
| :--- | :--- | :--- |
| `/track`, `/click` hot | Local Go filters, one `EVALSHA`, async stream, safe-page `/track/verify` cold-ish gnet route | PG/CH queries, ML inference, `fmt.Sprintf` in inner loops |
| Processor settlement | Batch PG, CH lookup with timeout, conversion mutation before outbox | Blocking tracker ingest |
| `cmd/control` workers | CH poll 15 min, outbox enqueue, webhook notify | Per-request work on tracker |
| Safe page JS | Client fingerprint on lander only | Sync render-blocking tag in redirect chain |
| Edge XDP | L3/L4 listed hosts | Residential rotation classification |

SLA reference: `core.mdc` - tracker p95 < 50 ms, p99 < 80 ms; Redis Lua p99 < 10 ms per shard.

---

## Summary

| Slug | Priority | ROI story | Rough surface | Est. effort |
| :--- | :--- | :--- | :--- | :--- |
| `conversion_smart_reject` | roi_p0 | Stop affiliate payouts on junk conversions | processor + postback + PG/CH | M |
| `automation_fraud_metrics` | roi_p0 | Auto-blacklist zones when IVT rate spikes | `internal/automation` + CH + admin UI | S-M |
| `canvas_noise_test_retest` | roi_p1 | Antidetect noise without IP | safe page JS + attestation | S |
| `cgnat_mobile_ip_policy` | roi_p2 | Cut CGNAT false positives on cellular | hot filters + GeoIP ASN | S |

---

## `conversion_smart_reject`

**Priority:** roi_p0

**Gap:** FraudScore SmartReject parity. Clicks can be filtered, but conversions/postbacks are accepted, paid out, and forwarded to CAPI unless `SilentRejectEvent`, `ShadowEvent`, or `FraudReason` already set on the event (`internal/postback/conversion_outbox.go` skips those).

**Current state:** `ConversionPayoutApplier` maps inbound status to `revenue_micro` only (`internal/ingestion/conversion_payout.go`). No TTI, click-existence, duplicate-goal, or IP-drift rules at settlement.

**Target:** Cold-path rejection before affiliate payout and before `ConversionPostbackEnqueuer.OnBatchStored` enqueues outbound CAPI/postback.

### Implementation

1. **Package:** add `internal/postback/conversion_reject.go` (or `internal/ingestion/conversion_fraud.go` if kept beside payout applier). Cold path only.

2. **Hook order** in processor settlement pipeline (`cmd/processor/main.go` wiring):
   - After event batch parsed, **before** `ConversionPayoutApplier.ApplyBatch` and **before** `WrapEventStoreAfterBatch` postback hook.
   - Mutate conversion `domain.Event`: set `FraudReason` (canonical code), optionally `SilentRejectEvent=true` per campaign policy.

3. **Rules v1** (campaign-configurable thresholds in PG or reuse fraud preset fields):

   | Rule | Signal code | Logic |
   | :--- | :--- | :--- |
   | Missing click | `conversion_no_click` | `click_id` empty or not found in settlement dedup / PG click row |
   | Fast conversion | `conversion_low_ttc` | `conversion_at - click_at < min_conversion_ttc_ms` (default 3000-5000 ms) |
   | Duplicate goal | `conversion_duplicate` | Same `campaign_id + click_id + goal_name` already in PG or CH conversions |
   | IP country drift | `conversion_ip_drift` | Click IP geo country != conversion IP country (not VPN-forgiving v1: strict Tier-1 only) |
   | DC at conversion | `conversion_datacenter_ip` | Reuse ASN snapshot / `datacenter_ip` tables at processor (not per-event HTTP) |

4. **Click lookup - avoid slop:**
   - **Do not** query ClickHouse synchronously per conversion in a tight loop (N+1 cold-path slop).
   - Batch: collect `click_ids` from conversion batch -> one `WHERE click_id IN (...)` on PG `clicks` or CH `clicks` with `context.WithTimeout` (reuse `coldpath` timeout pattern from automation CH queries).
   - Cache click row snapshot in a `map[string]clickSnapshot` for the batch only.

5. **Payout / ledger:**
   - Rejected conversions: `revenue_micro = 0` in CH insert; do not credit affiliate ledger lines (verify settlement lane skips revenue for `FraudReason != ""`).
   - Document in campaign fraud PATCH which rules are enabled.

6. **Outbound postback:** `ConversionPostbackEnqueuer` already skips `FraudReason != ""` - verify holdout after hook.

7. **Analytics:** insert rejected conversions into `fraud_events` or tag `conversions` table with `fraud_reason` column if missing (migration `internal/clickhouse/migrate/` + processor insert path). Reports must use `silent_reject_*` naming, not `ghost_*` (`anti-slop.mdc`).

8. **Admin API:** extend `PATCH /api/v1/campaigns/{id}/fraud` DTO with `conversion_reject_rules` JSON (enabled flags + thresholds). OpenAPI stub if bundle gate requires it.

9. **Tests (required):**
   - `internal/postback/conversion_reject_test.go` - table-driven rules
   - `internal/postback/conversion_outbox_test.go` - rejected conversion does not enqueue outbox
   - Integration: `conversion_reject_integration_test.go` with `testing.Short()` + `integration:` skip reason, real PG (`integration_test_slop_gate.sh`)

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Processor batch | One PG (or CH) round-trip per settlement batch, not per event |
| Tracker | **Zero** new work |

### Anti-slop gates

- [ ] No `_ = json.Unmarshal` in new handlers (`anti_slop_gate.sh`)
- [ ] No bare `t.Skip()` in integration tests
- [ ] Do not claim "legal proof" or Google refund eligibility in docs
- [x] Reason codes in `postback.ConversionRejectReasonWeights` (parallel registry; not hot-path `fraudReasonRegistry`)

### Done gates

- [x] `go test ./internal/postback/... -count=1` (unit)
- [x] Holdout: conversion with `conversion_low_ttc` -> no outbox row, `revenue_micro=0`
- [x] Pending validation: zero revenue, skip PG stats, skip postback until reprocess
- [x] `ANTIFRAUD.md` section: conversion rejection cold path
- [x] `conversion_reject_integration_test.go` (pending postback hold; Docker)
- [x] `go test ./internal/ingestion/ -run 'ConversionPayout|RollupCampaignStats' -count=1`
- [x] Admin UI: canvas retest + CGNAT toggles on fraud tab
- [x] Per-campaign `conversion_reject_rules` on fraud PATCH + admin UI
- [x] Unit verification: `go test` postback/ingestion/domain + `go build cmd/processor`

---

## `automation_fraud_metrics`

**Priority:** roi_p0

**Gap:** Automation rules support `roi_pct`, `cr`, `spend_micro`, `clicks` (`internal/automation/eval.go`) but not fraud-derived rates. Operators cannot auto-blacklist placements when IVT spikes.

**Current state:** `blacklist_placement` action exists (`internal/controlplane/automation_executor.go` -> `BlockCampaignPlacement`). CH source is `placement_stats_hourly` only (no fraud columns - see `internal/clickhouse/migrate/00005_placement_counts.sql`).

**Target:** Rules like `ivt_rate > 25` or `silent_reject_rate > 15` over 15-60 min window trigger `blacklist_placement` + optional `notify`.

### Implementation

1. **CH rollup (preferred):** new migration adding columns to `placement_stats_hourly` or sibling table `placement_fraud_stats_hourly`:
   - `fraud_reject_count`, `silent_reject_count`, `shadow_count`
   - Materialized views from `fraud_events` grouped by `campaign_id`, `placement_id`, hour
   - Avoid double-counting: align placement_id extraction with `reports_fraud.go` (`JSONExtractString(payload, 'placement_id')`)

2. **Alternative (faster v1, higher query cost):** second CH query in `automation/eval.go` joining `fraud_events` to placement metrics in Go. Acceptable for 15 min worker tick; **not** on hot path. Use `chQueryTimeout` (15 s).

3. **Metrics** in `observedMetric` (`internal/automation/eval.go`):
   - `ivt_rate` = `fraud_reject_count / click_count` (guard zero clicks)
   - `silent_reject_rate` = `silent_reject_count / click_count`
   - `fraud_reject_count` absolute threshold optional

4. **Admin UI** (`web/src/views/integrations/automation` or equivalent):
   - Metric dropdown includes new keys
   - Dry-run API shows observed values (`automation` dry-run endpoint)

5. **Docs:** `docs/INTEGRATIONS.md` automation section - list fraud metrics

6. **Tests:**
   - `internal/automation/metrics_test.go` - rate calculation edge cases (0 clicks)
   - `internal/automation/worker_integration_test.go` - fixture with fraud rows fires blacklist

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Automation worker | 15 min tick; CH query p99 < 15 s total per tick |
| Hot path | No change |

### Anti-slop gates

- [ ] Do not add automation metric that always returns 0 without CH fixture test
- [ ] `blacklist_placement` still via outbox (`control-plane.mdc`) - no direct Redis from worker

### Done gates

- [x] `go test ./internal/automation/... -count=1` (unit; integration needs Docker)
- [x] Example rule in UI or INTEGRATIONS.md: IVT rate -> blacklist
- [x] Cooldown respected (`Rule.CooldownMinutes`) on repeat fire
- [ ] CH migration applied in dev compose profile `analytics-ml` (v1 uses live `fraud_events` query; rollup MV optional follow-up)

---

## `canvas_noise_test_retest`

**Priority:** roi_p1

**Gap:** Antidetect browsers inject per-read Canvas noise. Current safe page checks hash presence and WebGL strings (`safe_page_attestation.go`) but not test-retest inconsistency.

**Current state:** `safe_page_hydrator.js` renders canvas once (`canvas_hash`). Attestation rejects missing/invalid hash and SwiftShader renderer.

**Target:** Two identical canvas draws in one page load; mismatch -> attestation failure -> L1 or safe-page deny before offer redirect.

### Implementation

1. **JS** (`internal/ingestion/safe_page_hydrator.js` - edit source; rebuild/minify if pipeline exists):
   - Function `canvasHashOnce()` - existing draw
   - Call twice; set `canvas_hash_a`, `canvas_hash_b` (or `canvas_retest_mismatch: true`)
   - Keep work async; do not block first paint longer than current script (measure: no new sync canvas in head before interaction)

2. **Go struct** `safePageVerifyFingerprint` (`safe_page_verify.go`): add `CanvasHashB string` or `CanvasRetestMismatch bool`

3. **Attestation** (`safe_page_attestation_advanced.go`):
   - New code `canvas_retest_mismatch` when both hashes non-empty and differ
   - Register weight in safe-page scoring path (maps to existing fraud layer on verify failure)

4. **Do not** add this signal to `/click` redirect-only path without safe page - document scope in `ANTIFRAUD.md`

5. **False positives:** Firefox/WebKit privacy noise - gate behind campaign flag `canvas_retest_enabled` default off; document fail-open when only one hash present

6. **Tests:**
   - `safe_page_attestation_advanced_test.go` - matching hashes pass; mismatch fails
   - `safe_page_attestation_test.go` - holdout: single hash does not false-positive when flag off

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| `/track/verify` | Extra ~1-5 ms client JS; server-side only compares two hex strings |
| `/click` redirect | **No** added JS |

### Anti-slop gates

- [ ] No marketing claim of fixed accuracy % in docs
- [ ] Do not cite safe-page bench as tracker p99 SLA

### Done gates

- [x] `go test ./internal/ingestion/ -run SafePage -count=1`
- [x] `ANTIFRAUD.md` - safe page antidetect row in signal table
- [x] Campaign fraud PATCH optional flag documented

---

## `cgnat_mobile_ip_policy`

**Priority:** roi_p2

**Gap:** IP velocity (`ipv4_rotation`) and ingress RPD treat CGNAT mobile gateways as botnets. Market analysis and `ANTIFRAUD_MARKET_ANALYSIS.md` section 9.2.

**Current state:** `IPv4RotationTable` on `/click` (`click_redirect.go`, `landing_ipv4_rotation_hook`). Ingress RPD in `EntitlementsFilter` + `checkIngressRPDGo` (`lua_precheck.go`). `mobileASNDenylist` in `dc_asn_table.go` only excludes specific ASNs from DC classification - no positive mobile-carrier allowlist for IP velocity bypass.

**Target:** When connection is classified as mobile carrier ASN, skip **only** IP-frequency signals (`ipv4_rotation`, ingress RPD keyed by IP). Keep `datacenter_ip`, TLS blocklist, attestation, `l3_blocklist`, ML boost.

### Implementation

1. **Mobile carrier ASN set:**
   - Extend GeoIP enricher or reuse `internal/edge/perimeter_asn_whitelist.go` mobile map pattern
   - Snapshot loaded at registry refresh (atomic pointer) - **no per-request file I/O**
   - Optional campaign `conn_type_policy` = `mobile_only` already exists - align semantics

2. **IPv4 rotation** (`click_redirect.go` / `IPv4RotationTable.Observe`):
   - Early return when `isMobileCarrierASN(evt.ASN)` and campaign flag `cgnat_ip_policy_enabled` (or global env `CGNAT_MOBILE_IP_BYPASS=1`)

3. **Ingress RPD** (`checkIngressRPDGo`):
   - When mobile carrier + flag: skip INCR for that event (document interaction with `SetIngressRPDHandledExternally`)

4. **Do not** disable:
   - `FraudBlacklistFilter` / `l3_blocklist`
   - `datacenter_ip` (mobile ASNs must not be in DC table)
   - `residential_proxy` intel
   - UnifiedFilter Lua budget path

5. **Metrics:** `ad_cgnat_ip_bypass_total{signal="ipv4_rotation|ingress_rpd"}` - fixed label set

6. **Tests:**
   - `TestCGNAT_mobileASN_skipsIPv4Rotation_holdout`
   - `TestCGNAT_mobileASN_stillBlocksDatacenter` - DC ASN on mobile policy still L1-high
   - Document mobile **proxy** ASNs remain blockable (do not whitelist all cellular resellers)

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Hot path | One map lookup on existing ASN field - 0 allocs if snapshot pointer |

### Anti-slop gates

- [x] Docs must not say "we ignore IP for all mobile traffic" - specify velocity/RPD only
- [x] Do not disable Redis INCR globally for mobile (bots use mobile proxies)

### Done gates

- [x] `go test ./internal/ingestion/ -run 'CGNAT|TestClickRedirect_IPv4Rotation|IngressRPD' -count=1`
- [x] `go test -short ./internal/ingestion/ -run 'ZeroAlloc|Check_zeroAlloc_localQuantaFullSkip' -count=1` (alloc gate subset; `lua_precheck.go` touched)
- [x] `ANTIFRAUD.md` CGNAT subsection updated
- [x] `deploy/vendor/ANTIFRAUD_MARKET_ANALYSIS.md` internal alignment note if claims change

---

## Suggested ship order (LLM codegen batches)

| Batch | Slug | Why first |
| :--- | :--- | :--- |
| 1 | `conversion_smart_reject` | Money on payouts; isolated cold path |
| 2 | `automation_fraud_metrics` | Uses existing blacklist action; CH + eval only |
| 3 | `canvas_noise_test_retest` | Small JS + attestation; clear tests |
| 4 | `cgnat_mobile_ip_policy` | Needs careful FP tuning; ship with flag default off |

Each batch: one PR, one commit surface name in title, scoped `pr_fast.sh`.

---

## Related activation (no new code - sales ROI)

These are already shipped; enable in pilot installs before building new slugs:

- [ ] Compose profile `analytics-ml` for Pro SKU `ivt_ml_detector`
- [ ] Campaign wizard default `enhanced_defense` preset (`service_fraud_enhanced_defense.go`)
- [ ] Click log + fraud breakdown reports for operator disputes (`reports_click_log.go`, `reports_fraud.go`)
- [ ] Margin guard policies on high-spend campaigns (`internal/ledger/worker.go`)

---

## Verification commands (orchestrator)

```bash
go test ./internal/ingestion/ -run 'Fraud|SafePage|TLS|SilentReject|IPv4Rotation' -count=1
go test ./internal/postback/... -count=1
go test ./internal/automation/... -count=1
bash scripts/ci/antifraud_doc_gate.sh
bash scripts/ci/anti_slop_gate.sh
bash scripts/ci/pr_fast.sh
```

Hot-path edits additionally:

```bash
make test-alloc-gate
bash scripts/ci/hot_path_static_gate.sh
bash scripts/ci/escape_heap_gate.sh
```
