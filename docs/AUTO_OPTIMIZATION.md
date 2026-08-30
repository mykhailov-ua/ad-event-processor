# Auto-Optimization (traffic weight rules)

Engineering spec for a **separate cold-path feature**: operator-configurable rules that periodically recompute **lander / offer / brand-creative weights** from ClickHouse stats, using objectives **CR, EPC, ROI, or revenue**. Hot path only **reads precomputed weights** from the existing flow snapshot (no new I/O per `/track`).

Constraints: `architecture.mdc`, `hot-path.mdc`, `anti-slop.mdc`, `modular-monolith.mdc`, `testing.mdc`.

**Not in scope for v1:** per-request ML, Postgres/CH on `/track`, dynamic Prometheus labels, rewriting `automation` guard rules into weight shifts.

---

## Current baseline (do not re-ship as the product)

| Piece | Location | Behavior today |
| :--- | :--- | :--- |
| Flow bandit | `internal/flow/bandit_tx.go`, `bandit_thompson.go` | Thompson sampling on **CR** (clicks + conversions); min 100 clicks; env-only via `DeliveryOptimizer` + `MAB_INTERVAL_MS` |
| Brand creative MAB | `internal/campaign/worker/mab.go` | **CTR** weights from impressions/clicks |
| Hot-path select | `internal/filter/flow_routing.go` `SelectSnapshot` / `BanditSelect` | Weighted pick from `FlowPathSnapshot`; **0 allocs/op** (`TestBanditSelect_ZeroAlloc`) |
| Automation | `internal/automation` | **Pause / blacklist / notify** on ROI, CR, spend, fraud — not weight optimization |
| Delivery optimizer tick | `internal/campaign/worker/delivery_tick.go` | Pacing + autoscale + optional MAB in one PG txn |

Auto-Optimization **unifies and productizes** weight updates behind explicit rules; it does **not** add a second selection algorithm on the hot path.

---

## Architecture invariants

### Hot path (`cmd/tracker`, `internal/ingest`, `internal/filter`)

| Rule | Requirement |
| :--- | :--- |
| I/O | **Zero** sync Postgres, ClickHouse, Redis eval, or HTTP on weight selection |
| Algorithm | Keep `selectWeightedLander` / `selectWeightedOffer`; only **weights** in snapshot change |
| Allocations | `BanditSelect` / flow select stays **0 allocs/op** on touched benchmarks |
| Config | Campaign/flow weights arrive via existing **pub/sub + atomic snapshot** reload |
| Forbidden | `fmt.Sprintf`, `interface{}`, `context.With*` in select path; per-event CH queries |

### Cold path (`cmd/control`)

| Rule | Requirement |
| :--- | :--- |
| Package | New domain `internal/trafficoptimizer/` (handlers, store, worker, eval) + thin `trafficoptimizer_bridge.go` in controlplane |
| Stats | ClickHouse only (`placement_stats_hourly`, clicks/conversions payloads, brand stats) — same pattern as `automation/eval.go` |
| Writes | PG update `flows.paths` / `brand_creatives.weight` + **outbox** `CAMPAIGN_UPDATE` / brand creative reload — one writer per entity per tick |
| Tick | Worker interval env floor (default 15 min); per-rule `eval_interval_minutes` like automation |
| Idempotency | `traffic_optimizer_fires` (or reuse pattern from `automation_rule_fires`) with `action_hash` |

### Data model (sketch)

```
traffic_optimizer_rules
  id, customer_id, campaign_id?, flow_id?, scope ENUM(lander|offer|creative)
  objective ENUM(cr|epc|roi|revenue)
  lookback_minutes, min_clicks, min_conversions, min_spend_micro
  eval_interval_minutes, cooldown_minutes, enabled
  algorithm ENUM(thompson|proportional)  -- v1: proportional for epc/roi/revenue; thompson for cr only
  max_weight_delta_pct  -- cap per tick (exploration guard)
```

Presets mirror automation: `GET /api/v1/traffic-optimizer/presets`.

---

## Rollout

### Contract baseline (no behavior change)

**Status:** shipped (2026-08-30).

**Goal:** Named feature, API sketch, metrics, test plan; map existing MAB to future rule types.

| Deliverable | Detail |
| :--- | :--- |
| OpenAPI | `paths/traffic_optimizer.yaml`: presets, rules CRUD, dry-run |
| PG migration | Tables above + indexes on `(customer_id, enabled)` |
| Metrics | `traffic_optimizer_eval_total`, `traffic_optimizer_weight_updates_total`, `traffic_optimizer_last_tick_seconds` |
| Doc | This file + row in `docs/DOCS.md` |
| Refactor spike (optional) | `trafficoptimizer.FlowWeightApplier` delegates to `flow.ApplyFlowBanditThompson` (`applier.go`) |

**Shipped artifacts**

| Path | Role |
| :--- | :--- |
| `internal/trafficoptimizer/` | Domain package: presets, handlers, metrics, `RulesService` (list read-only until writes enabled; full CRUD when worker enabled) |
| `internal/ingest/migrations/00127_traffic_optimizer_rules.sql` | PG tables |
| `internal/ingest/queries/traffic_optimizer.sql` | sqlc list/get queries |
| `api/openapi/paths/traffic_optimizer.yaml` | Documented routes |
| `internal/controlplane/trafficoptimizer_bridge.go` | Wire |

**Exit criteria**

- `bash scripts/ci/openapi_gate.sh` green for new paths (stub handlers 501 OK).
- No tracker import of `internal/trafficoptimizer`.

---

### CR objective (flow bandit) — **shipped**

**Goal:** Operator-visible rules for **lander/offer** weight shifts by **CR**, replacing implicit env-only MAB for flows.

| Deliverable | Detail |
| :--- | :--- |
| Worker | `trafficoptimizer.Worker` tick: load rules → CH stats → `ApplyFlowBanditThompson` → PG + outbox |
| API | Create/list/update/delete rules; preset `cr_best_performer` |
| Delivery delegate | `TRAFFIC_OPTIMIZER_ENABLED=1` skips `flow.OptimizeFlowBanditTx` in delivery tick |
| Feature flag | `TRAFFIC_OPTIMIZER_ENABLED`, `TRAFFIC_OPTIMIZER_INTERVAL_MIN` (floor 5–60) |

**Shipped artifacts**

| Path | Role |
| :--- | :--- |
| `internal/trafficoptimizer/worker.go` | Rule tick, cooldown, outbox publish |
| `internal/trafficoptimizer/optimize_tx.go` | `ApplyRuleTx` + idempotent `traffic_optimizer_fires` |
| `internal/trafficoptimizer/rules_service.go` | CRUD + CR-scope validation |
| `internal/flow/bandit_tx.go` | Filtered bandit (`BanditFlowFilter`, `BanditApplyConfig`) |
| `internal/campaign/worker/delivery_tick.go` | Skip legacy flow MAB when optimizer enabled |
| `internal/controlplane/trafficoptimizer_bridge.go` | Worker host + `StartTrafficOptimizerWorker` |

**Exit criteria**

- Unit/holdout: `go test ./internal/trafficoptimizer/ -short` (`TestApplyRuleTx_holdout_*`, `TestRulesService_buildRuleParams_*`).
- Integration (tier 2, not in `-short`): CH fixture → weight change — `TestTrafficOptimizer_CR_EndToEnd` in `internal/controlplane/`.
- `TestFault_DeliveryOptimizerSingleWriter` still passes (≤1 outbox event per campaign per tick).
- Hot path: no tracker import of `internal/trafficoptimizer`.

---

### EPC and revenue objectives — **shipped**

**Goal:** Rules optimize on **EPC** (`payout / clicks`) or **total revenue** in window (proportional weights, not Thompson).

| Deliverable | Detail |
| :--- | :--- |
| Algorithm | `flow.ProportionalWeights` + `ApplyFlowBanditProportional` with `max_weight_delta_pct` clamp |
| Presets | `epc_best_performer`, `revenue_best_performer` |
| Validation | Objective/algorithm pairing; `min_clicks` ≥ 100; lookback > 7d gated by `AllowExtendedLookback` |

**Shipped artifacts**

| Path | Role |
| :--- | :--- |
| `internal/flow/bandit_proportional.go` | EPC/revenue scoring + weight clamp |
| `internal/flow/bandit_tx.go` | Proportional apply path + `BanditApplyConfig` extensions |
| `internal/trafficoptimizer/validate.go` | EPC/revenue + proportional validation |
| `internal/trafficoptimizer/optimize_tx.go` | Rule dispatch for CR and proportional objectives |

**Exit criteria**

- Unit: `go test ./internal/flow/ ./internal/trafficoptimizer/ -short` (`TestProportionalWeights_*`, `TestProportionalWeights_holdout_invertedSort`).
- Integration (tier 2): synthetic CH rows → expected weight ordering — `TestTrafficOptimizer_ROI_EndToEnd` (proportional ROI); CR path in `TestTrafficOptimizer_CR_EndToEnd` (Thompson).
- Holdout: `TestRuleSupported_holdout_epcRequiresProportional`, inverted sort holdout in `flow`.

---

### ROI objective and brand creatives scope — **shipped**

**Goal:** **ROI** (`(revenue - spend) / spend`) on lander/offer paths and **brand creatives** via proportional weights.

| Deliverable | Detail |
| :--- | :--- |
| Spend | `spend_micro` from click payload in CH bandit queries |
| Creative scope | `scope=creative` + `brand_id`; updates `brand_creatives.weight` + brand outbox |
| Preset | `roi_best_performer` (override `scope=creative` + `brand_id` for creatives) |
| Deprecation | `TRAFFIC_OPTIMIZER_ENABLED=1` skips legacy `OptimizeBrandCreativeMABTx` in delivery tick |

**Shipped artifacts**

| Path | Role |
| :--- | :--- |
| `internal/flow/bandit_proportional.go` | `BanditObjectiveROI` scoring |
| `internal/flow/creative_bandit.go` | Creative attribution + proportional apply |
| `internal/trafficoptimizer/creative_tx.go` | `ApplyCreativeRuleTx` |
| `internal/controlplane/creative_bandit_bridge.go` | CH revenue/spend for creatives |

**Exit criteria**

- Unit: `go test ./internal/flow/ ./internal/trafficoptimizer/ -short`.
- Integration: ROI with spend + revenue in CH — `TestTrafficOptimizer_ROI_EndToEnd` in `internal/controlplane/` (tier 2, not `-short`).
- Budget invariant on weight-only tick — covered by `TestTrafficOptimizer_ROI_EndToEnd` (`AssertBudgetInvariant`).

---

### Operator UX and dry-run

**Goal:** Safe rollout: preview weights before apply.

| Deliverable | Detail |
| :--- | :--- |
| API | `POST /api/v1/traffic-optimizer/rules/{id}/dry-run` → proposed weights, observed metrics per arm, no PG write |
| Audit | `traffic_optimizer_fires` history with before/after weights JSON |
| Admin route | `/integrations/traffic-optimizer` when `web/` returns (`ui.mdc`) |

**Exit criteria**

- Dry-run matches apply path math (shared `eval` function).
- RBAC: same campaign scope as automation rules.

---

### Hardening and load

**Goal:** Production SLOs and chaos.

| Deliverable | Detail |
| :--- | :--- |
| Caps | `TRAFFIC_OPTIMIZER_MAX_RULES_PER_CUSTOMER`, `MAX_EVALS_PER_TICK` (mirror automation) |
| CH guard | 15s query timeout per eval batch; skip customer on timeout (degraded metric, no partial PG write) |
| Fault | Double tick concurrent → single writer / serializable txn |
| Load | Parser/load tests: **no** regression on control-cohort `/track` p99 |

**Exit criteria**

- `bash scripts/fault/run.sh` includes traffic optimizer tick under fault matrix (or dedicated `*_fault_test.go`).
- `traffic_optimizer_last_tick_seconds` < degraded threshold (600s) in ops health.

---

## Testing requirements

### Tier honesty (`anti-slop.mdc`)

| Claim | Minimum tier | Command |
| :--- | :--- | :--- |
| Rule CRUD / validation | test-fast | `go test ./internal/trafficoptimizer/... -short -count=1` |
| CH → weight → snapshot | test-integration | `make test-integration` (subset `-run TrafficOptimizer`) |
| Single writer / concurrent tick | test-fault | `make test-fault` or `TestFault_TrafficOptimizerSingleWriter` |
| Hot path unaffected | test-alloc-gate | `make test-alloc-gate` |
| E2E `/track` p99 | load / BPF | `malformed.sh` control cohort — not unit bench |

**Never cite** `BenchmarkUnifiedFilter_Check_mock` or mock Redis as proof of optimizer behavior.

### Mandatory holdouts

| Behavior | Test name (required) | Fails if |
| :--- | :--- | :--- |
| Weight apply removed | `TestTrafficOptimizer_apply_holdout` | Weights stay at seed |
| Proportional sort inverted | `TestTrafficOptimizer_proportionalSort_holdout` | Best arm gets lowest weight |
| Dry-run vs apply drift | `TestTrafficOptimizer_dryRunParity_holdout` | Different math in dry-run |
| Hot path alloc regression | `TestBanditSelect_ZeroAlloc` | Any alloc in select |
| Double outbox per campaign | `TestFault_TrafficOptimizerSingleWriter` | >1 campaign update event per tick |
| Budget unchanged | `TestTrafficOptimizer_noSpendSideEffect_holdout` | `current_spend` drifts on weight-only tick |

### Integration test contract (`integration_test_slop.sh`)

Every `*_integration_test.go`:

1. `testing.Short()` skip with `integration:` prefix.
2. Real `SetupPostgres` + ClickHouse testcontainer (or shared merge-integration harness).
3. Behavioral assertions on **weight values** and optional pub/sub snapshot read — not `err == nil` only.
4. No `testify/mock` for CH/PG.

### Unit tests (keep)

- Objective math: CR, EPC micro, ROI %, revenue tie-break.
- `max_weight_delta_pct` clamp.
- Preset expand / parameter merge (mirror `automation/presets.go`).
- JSON path patch roundtrip on `flows.paths`.

### Banned tests

- Tautology `require.Equal(t, w, w)`.
- Table that only asserts mock return values.
- Chaos with `pct=0`.
- Claiming optimizer works from `httptest` handler without CH fixture.

---

## Performance and SLA

### Tracker ingest (unchanged budgets — `core.mdc`)

| Metric | Ceiling | Verification |
| :--- | :--- | :--- |
| `ad_http_request_duration_seconds` p95 | < 50 ms | Load test / Prometheus |
| p99 | < 80 ms (hard 100 ms) | `malformed.sh` abort if control cohort > 80 ms for 30s |
| Flow select | 0 allocs/op | `TestBanditSelect_ZeroAlloc`, `make test-alloc-gate` |
| New heap escapes in hot files | 0 vs baseline | `bash scripts/ci/static/escape_heap.sh` |

Weight updates are **async**. Staleness: weights may lag CH by up to `eval_interval_minutes` + outbox + pub/sub (document in API `stale_weights` hint on dry-run).

### Cold worker

| Knob | Default | Limit |
| :--- | :--- | :--- |
| `TRAFFIC_OPTIMIZER_INTERVAL_MIN` | 15 | 5–60 |
| Per-rule `eval_interval_minutes` | 15 | ≥ global floor |
| CH query timeout | 15s | Same as automation |
| `min_clicks` (flow arms) | 100 | Do not lower without holdout review |
| `max_weight_delta_pct` | 50 | Prevents 100→1 swings in one tick |

| Metric | Degraded | Critical |
| :--- | :--- | :--- |
| `traffic_optimizer_last_tick_seconds` | > 600 | > 3600 |
| CH eval errors / tick | > 0 sustained | Worker stuck > 1h |

### ClickHouse cost

- Batch campaigns per customer per tick (cap `MAX_EVALS_PER_TICK`).
- Prefer `placement_stats_hourly` + bounded payload extracts; no full `clicks` table scan per rule.
- Index-friendly filters: `campaign_id`, `hour` range.

---

## API surface (target)

| Method | Path |
| :--- | :--- |
| GET | `/api/v1/traffic-optimizer/presets` |
| GET/POST | `/api/v1/traffic-optimizer/rules` |
| PUT/DELETE | `/api/v1/traffic-optimizer/rules/{id}` |
| POST | `/api/v1/traffic-optimizer/rules/{id}/dry-run` |

Rule body fields: `scope`, `objective`, `campaign_id`, optional `flow_id` / `brand_id`, thresholds, `algorithm`, `enabled`.

License: optional SKU flag `traffic_optimizer` in `sku.yaml` when product gates multi-objective or creative scope.

---

## Verification matrix (per PR)

| Touched surface | Run |
| :--- | :--- |
| `internal/trafficoptimizer/` (cold) | `go test ./internal/trafficoptimizer/... -short -count=1` |
| Worker + outbox | `go test ./internal/controlplane/ -run TrafficOptimizer -count=1` + integration tier if CH |
| `flow_routing.go` / ingest select | `make test-alloc-gate`; `escape_heap.sh` |
| OpenAPI | `bash scripts/ci/openapi_gate.sh` |
| Default merge gate | `bash scripts/ci/pr_fast.sh` |
| Weight + CH wiring claim | `make test-integration` (paste exit code in PR) |
| Fault / single writer | `make test-fault` or named `*_fault_test.go` |

---

## PR checklist (agents)

1. Hot path: no new imports from `trafficoptimizer`; snapshot read only.
2. Name verification **tier** in PR body (`test-fast` vs `integration` vs `fault`).
3. Add holdout that fails if weight apply reverted.
4. Cite `TestFault_DeliveryOptimizerSingleWriter` when touching delivery tick outbox.
5. Do not weaken `TestBanditSelect_ZeroAlloc` or alloc gate baselines.
6. Doc claims in this file updated in the **same PR** as behavior changes (`anti-slop.mdc` doc lie mode).

---

## Related code

- Flow select: `internal/filter/flow_routing.go`
- Existing bandit: `internal/flow/bandit_tx.go`, `internal/controlplane/campaign_bandit_bridge.go`
- Delivery tick: `internal/campaign/worker/delivery_tick.go`
- Automation pattern: `internal/automation/`, `docs/INTEGRATIONS.md` (Campaign automation rules)
- Tests: `internal/ingest/flow_router_test.go`, `internal/controlplane/delivery_optimizer_fault_test.go`
