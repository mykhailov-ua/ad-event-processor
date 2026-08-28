# Modular structure backlog

Repository layout parity with **lead-intent-processor** as a **quality reference**, not a 1:1 copy. Lead-intent is a single-pipeline monolith (`internal/app` + `sources/*` + `warmpath`/`coldpath`); ad-event-processor is a multi-binary ad platform with hot ingest, broker, RTB, and a cold admin plane. The goal is the same **invariants**: thin composition root, domain-owned logic, `pkg/` without `internal/*`, no package above ~40 production `.go` files.

**Reference repo:** `lead-intent-processor` (`internal/app`, `internal/sources/*`, `pkg/bpfenv` only).

**Canonical rules:** `structure.mdc`, `modular-monolith.mdc`, `boundaries.mdc`, `naming.mdc`.

**Out of scope:** product features ([competitive_backlog.md](./competitive_backlog.md)), admin UI ([admin_ui_redesign_backlog.md](./admin_ui_redesign_backlog.md)), antifraud semantics ([ANTIFRAUD.md](./ANTIFRAUD.md)).

Cross-reference slugs in PR descriptions. Do not mark a slug closed until done gates pass in the same commit as code.

---

## Reference comparison

| Invariant | lead-intent-processor | ad-event-processor (2026-08-28) | Target |
| :--- | :--- | :--- | :--- |
| Composition root size | `internal/app` ~18 prod `.go` | `internal/controlplane` ~30 | `controlplane` <= 40 prod `.go` (wire + bridges + shell only) |
| Type re-exports | None | 19 `*_aliases.go` | 0 — direct imports |
| `pkg/` role | 1 package (`bpfenv`), zero `internal/*` | 24 packages; single-consumer merges done | Shared util only; 0 prod `pkg` -> `internal` imports |
| Largest domain package | `sources/` ~60 `.go` (tree) | `ingestion` ~297 in one package | Hot path split; no package > 40 prod `.go` |
| Path temperature in tree | `ingest`, `warmpath`, `coldpath` | Mostly `ingestion` + scattered cold packages | Explicit hot/cold folder names (see target tree) |
| Admin domains | `crm/{admin,store,webhook,app}` | `*admin` partial + `service_*` still in controlplane | Logic in domain packages; bridges only |
| Wiring | `app/deps.go` `buildDeps()` | `adminapi_wire.go` + 27 `*_bridge.go` | Bridges shrink as domains move out |

---

## Ceilings (merge blockers after P1)

From `modular-monolith.mdc`:

| Surface | Ceiling |
| :--- | :--- |
| Package prod `.go` count | **40** (exclude `db/`, `pb/`, `queries/`, `*_test.go`) |
| Package LOC | **~8000** non-generated |
| Single prod file | **~500** lines |
| Domain port (`Host`, `Effects`, …) | **<= 12** methods |
| `*_bridge.go` | **<= 200** lines |
| `controlplane` net LOC | **Must shrink** on each domain extraction PR |

---

## Layout rules (flat packages + file names)

Reference: lead-intent uses `internal/warmpath/service.go`, not `internal/warmpath/server/service.go`. Path = package boundary; bridges handle cross-package calls.

| Rule | Detail |
| :--- | :--- |
| No redundant subdirs | Ban `internal/<domain>/server/`, `internal/<domain>/admin/` when parent already is the domain. Use `internal/broker/`, `internal/regionproxy/`, not `.../server/`. |
| `pkg/` vs `internal/` | `pkg/broker/{client,protocol,log,consumer}` = wire protocol (no `internal/*`). `internal/broker/` = gnet daemon. Cross-call via import or `controlplane` bridge — not a third folder. |
| Domain file names | Inside `internal/<domain>/`: role only (`handlers.go`, `decisions.go`). No `<domain>_` prefix (`fraudadmin_handlers.go` banned). |
| `controlplane` bridges | `*_bridge.go` may keep domain token — wiring only, <= 200 lines. |
| **Banned** | `*_aliases.go`, new `service_<domain>_*.go`, `domains.go` prefix tokens for new domains |
| Ban `service_*` in controlplane | After drain: no new `service_<domain>_*.go`; logic in domain package, bridge wires `Host`. |
| Classify by package path | `composition_drain_inventory.go` + CI package path — not filename prefix lists |

**Filename hygiene slug:** `domain_filename_hygiene` (P1) — rename matrix per domain after each drain PR; CI gate in P4 `structure_ci_gate_package_size` extended with `lint_go_filename_gate.sh`.

---

## Summary table

| Slug | Priority | Area | Status |
| :--- | :--- | :--- | :--- |
| `pkg_boundary_hygiene` | P0 | `pkg/` | shipped |
| `remove_pkg_duplicate_servers` | P0 | `pkg/broker` | shipped |
| `composition_root_inventory` | P0 | `controlplane` | shipped |
| `flatten_broker_package` | P1 | `internal/broker` | shipped |
| `flatten_regionproxy_package` | P1 | `internal/regionproxy` | shipped |
| `reports_fraud_subtree_stable` | P1 | `reports` | shipped |
| `drain_fraud_service_to_fraudadmin` | P1 | `fraudadmin` | shipped |
| `drain_rtb_service_to_rtbadmin` | P1 | `rtbadmin` | shipped |
| `drain_license_service_to_licensingadmin` | P1 | `licensingadmin` | shipped |
| `drain_dashboards_to_dashboardadmin` | P1 | `dashboardadmin` | shipped |
| `drain_platform_team_to_platformadmin` | P1 | `platformadmin` | shipped |
| `drain_billing_service_to_billingadmin` | P1 | `billingadmin` | shipped |
| `outbox_domain_final_cut` | P1 | `outbox` | shipped |
| `domain_filename_hygiene` | P1 | all `internal/*` | shipped |
| `controlplane_wire_only_shell` | P1 | `controlplane` | open |
| `remove_controlplane_aliases` | P1 | `controlplane` | open |
| `ingestion_split_phase_track` | P2 | hot path | open |
| `ingestion_split_phase_filter` | P2 | hot path | open |
| `ingestion_split_phase_stream` | P2 | hot path | open |
| `ingestion_split_phase_ingest` | P2 | hot path | open |
| `campaign_subpackages_runtime_worker` | P2 | `campaign` | open |
| `licensing_package_split` | P3 | `licensing` | open |
| `domain_package_split` | P3 | `domain` | open |
| `cmd_deps_pattern` | P3 | `cmd/*` | open |
| `structure_ci_gate_package_size` | P4 | CI | open |

**Suggested ship order:** P0 (done) -> `flatten_*` (done) -> `reports_fraud_subtree_stable` -> `remove_controlplane_aliases` (as call sites migrate) -> `controlplane_wire_only_shell` gate -> P2 ingestion phases.

---

## P1 — remove alias and prefix cruft

### `remove_controlplane_aliases`

**Priority:** P1 (parallel with `controlplane_wire_only_shell`)

**Gap:** 19 `controlplane/*_aliases.go` files re-export domain types; `domains.go` classifies ownership by filename prefix (`service_fraud`, `node_`, …). `lead-intent-processor` has zero alias files — `internal/app` imports domain packages directly.

**Target:**

- Delete each `controlplane/*_aliases.go` when last `controlplane.Foo` reference becomes `domain.Foo`
- Delete misplaced `internal/nodeadmin_aliases.go` (wrong directory; `package controlplane`)
- Replace `domains.go` `Prefixes` rows with package-path rules in `composition_drain_inventory.go`
- Tests: import `campaign`, `fraudadmin`, etc. directly — not `controlplane.CampaignDTO`

**Forbidden after this slug starts:**

- New `*_aliases.go`
- New pass-through wrappers whose only job is re-export
- New `domains.go` prefix tokens for classification

**Done gates:**

- [ ] `find internal/controlplane -name '*_aliases.go' | wc -l` = 0
- [ ] `test ! -f internal/nodeadmin_aliases.go`
- [ ] `go test ./internal/controlplane/... -short -count=1`
- [ ] No new references to `controlplane.<Domain>Type` where `<Domain>` lives in `internal/<domain>/`

---

## Target tree (end state)

Not a rename of lead-intent folders; same **roles**:

```
cmd/
  tracker/main.go          # config + wire only
  processor/main.go
  broker/main.go
  control/main.go          # -> internal/control (runner), not business logic
  ...

internal/
  controlplane/            # <= 40 files: adminapi_wire, bridges, Service shell, workers registry
    doc.go
    adminapi_wire.go
    service.go             # deps struct, lazy inits, no domain HTTP bodies
    workers.go
    *_bridge.go            # wiring only, <= 200 lines each
    authz/
    admin_static_stub/

  # Hot path (lead-intent: pipeline + filter + sink)
  ingest/                  # gnet accept, HTTP/1 decode budget, conn lifecycle
    doc.go
    handler.go
    conn_*.go
    parser/
  track/                   # /track, /click, macros, attribution
    doc.go
    track_core.go
    click_*.go
  filter/                  # FilterEngine, local filters, fraud boost snapshot read
    doc.go
    engine.go
    filters.go
    unified_*.go
  stream/                  # producers, admission, broker defer, rollback
    doc.go
    producer.go
    broker_producer.go
    admission.go
  ingestion/               # TEMP: shrink to glue re-exports until callers migrated; then delete

  # Cold admin domains (lead-intent: crm/*)
  campaign/
    doc.go
    handlers.go
    runtime.go
    worker/                # delivery, pacing, MAB ticks
    wizard/
    editor/
  fraudadmin/ ...
  rtbadmin/ ...
  dashboardadmin/ ...
  platformadmin/ ...
  billingadmin/ ...
  opsadmin/ ...
  licensingadmin/ ...
  reports/
    doc.go
    catalog.go
    handlers.go
    fraud/                 # customer fraud CH queries (stable subfolder)
  outbox/
  reconciliation/
  governance/

  # Infra (flat — no /server subdir)
  broker/                  # gnet WAL broker daemon (package broker); pkg/broker = client/protocol only
  brokerreplay/
  regionproxy/             # gnet regional ingress; pkg/regionproxy = client/wal/keygen
  pgfailover/
  doctor/

  domain/                  # split: types | budget | shard (P3)
  licensing/               # split: verify | entitlements | trial (P3)

pkg/                       # lead-intent: bpfenv-only scope, but more pkgs allowed
  bpfenv/                  # (future) if BPF helpers move out of cmd
  gnetutil/
  lifecycle/
  piihash/
  coldpath/
  broker/                  # client, protocol, log, consumer ONLY (no server)
  regionproxy/             # client, wal, keygen, opkey, uplink, quorum, ingress/
  ...                      # no internal/* imports in prod .go
```

**Import matrix (unchanged):** `ingest|track|filter|stream` must not import `controlplane` or cold `internal/fraud` scoring. `pkg/*` must not import `internal/*` in production code.

---

## Global close checklist (every slug)

- [ ] `go build -o /dev/null` on touched `cmd/*` and `internal/*` packages
- [ ] `rg 'ad-event-processor/internal' pkg/ --glob '*.go' --glob '!*_test.go'` returns empty after P0
- [ ] Package file count for touched root package <= 40 (or waiver names follow-up slug in same release)
- [ ] Bridge file touched -> net LOC of matching `*_bridge.go` not increased without domain method count drop
- [ ] `python3 scripts/dev/sync_package_doc_go.py` run when adding packages; every new dir has `doc.go`
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `deploy/vendor/` prose
- [ ] PR cites slug; no `GAP-*` / `M-*` tokens in title

---

## P0 — boundary hygiene (do first)

### `pkg_boundary_hygiene`

**Priority:** P0

**Gap:** Production `pkg/*.go` still imports `internal/config`, `internal/metrics`, etc. Lead-intent has zero such imports.

**Target files:**

| Package | Change |
| :--- | :--- |
| `pkg/lifecycle/lifecycle.go` | `TimeoutsFromEnv()` reads `SHUTDOWN_TIMEOUT_MS` / `WAIT_TIMEOUT_MS` locally |
| `pkg/piihash/hasher.go` | `NewFromSalt(version, saltHex, fallbackSecret)`; callers pass fields from `config` |
| `pkg/regionproxy/client/client.go` | Import `pkg/regionproxy/ingress` for topic name, not `internal/regionproxy/server` |

**Done gates:**

- [x] `rg 'ad-event-processor/internal' pkg/ --glob '*.go' --glob '!*_test.go'` is empty
- [x] `bash scripts/ci/pkg_boundary_gate.sh`
- [x] `go test ./pkg/lifecycle/... ./pkg/piihash/... -short -count=1`

---

### `pkg_single_consumer_merge`

**Priority:** P1

**Gap:** Six `pkg/` trees had fan-out 0–2 and duplicated logic already owned by one `internal/<domain>/` package.

**Merged (shipped 2026-08-28):**

| From `pkg/` | To `internal/` |
| :--- | :--- |
| `gtax` | `domain/ctv_gtax_settlement.go` |
| `cpuset` + `runtimeautotune` | `ingestion/cpuset.go`, `ingestion/runtime_autotune.go` |
| `vendorprobe` | `platformadmin/vendor_probe.go` |
| `campaignmacro` | `campaign/macro_expand.go` |
| `bandit` | `flow/bandit_thompson.go` |

**Done gates:**

- [x] `test ! -d pkg/{gtax,cpuset,runtimeautotune,vendorprobe,campaignmacro,bandit}`
- [x] `go build ./...`
- [x] `bash scripts/ci/pkg_boundary_gate.sh`

---

### `pkg_tier_c_inventory`

**Priority:** P1 (document + gate; no further merges without new fan-out audit)

**Role:** Intentional `pkg/` ↔ `internal/` splits or multi-domain shared libs. **Do not merge** without cross-domain import count review.

| Package / pair | Importers (approx) | Role |
| :--- | ---: | :--- |
| `pkg/broker` ↔ `internal/broker` | 48 | Wire client/protocol/log/consumer vs gnet daemon (`cmd/broker`) |
| `pkg/regionproxy` ↔ `internal/regionproxy` | 10 | Client/wal/keygen/quorum vs gnet server (`cmd/region-proxy`) |
| `pkg/money` | 32 | Micro-unit money across billing, campaign, ledger |
| `pkg/coldpath` | 181 | Cold HTTP helpers across admin domains |
| `pkg/clientip` | 2 | Hot+cold client IP extraction (`ingestion`, `identity`) |
| `pkg/dedupkey` | 33 | Idempotency keys across admin + payment |
| `pkg/landerhost` | 10 | Flow delivery + hosted lander URLs |
| `pkg/domainhealth` | 6 | Integration health probes (supply, platformadmin, controlplane) |

**Done gates:**

- [x] `test ! -d pkg/broker/server pkg/regionproxy/server`
- [x] `bash scripts/ci/pkg_boundary_gate.sh`
- [x] `go build -o /dev/null ./cmd/broker/ ./cmd/region-proxy/`

---

### `model_layout_contract`

**Priority:** P1

**Gap:** Flat `model/*.py` mixed contract, train, eval, and IO without role boundaries.

**Target tree:**

| Subpackage | Role | Mirrors |
| :--- | :--- | :--- |
| `model/contract/` | `feature_spec`, `scoring_policy`, `policy_config`, parity fixtures | `internal/fraud/*.go` |
| `model/train/` | bootstrap, datasets, calibration, fixture generator | cold train only |
| `model/eval/` | shadow precision, simulation benchmark, PG eval store | offline metrics |
| `model/data/` | ClickHouse client, feature export, manual labels CSV | CH/PG IO |
| `model/repo_paths.py` | `REPO_ROOT`, fixture dirs | shared paths |
| `model/tests/`, `model/testdata/` | pytest + corpora | unchanged |

**Entrypoints:** `python3 -m train.artifact_bootstrap`, `train.fixture_generator`, `eval.evaluate`, `data.features_export` with `PYTHONPATH=model`.

**Done gates:**

- [x] `cd model && python3 -m pytest tests/ -q`
- [x] `PYTHONPATH=model python3 -m train.fixture_generator`
- [x] `go test ./internal/fraud/ -short -run 'TestFeatureSpec|TestScoringPolicyParity|TestPolicyConfigParity' -count=1`
- [x] `scripts/ci/fraudtrain.sh` uses `-m` entrypoints

---

### `pkg_tier_d_inventory`

**Priority:** P1 (document; shared layer — merge only with explicit slug)

**Role:** Cross-cutting utilities imported by many binaries. Stable `pkg/` surface.

| Package | Importers (approx) | Role |
| :--- | ---: | :--- |
| `pkg/lifecycle` | 19 | Graceful shutdown (`cmd/*`) |
| `pkg/piihash` | 27 | Salted PII hashing |
| `pkg/platformconfig` | 25 | Platform feature flags file |
| `pkg/naming` | 26 | Legacy naming CI helpers |
| `pkg/iogate` | 24 | PG/CH concurrency gates |
| `pkg/logger` | 13 | slog setup |
| `pkg/netaddr` | 15 | Listen address normalization |
| `pkg/branding` | 28 | White-label hosts |
| `pkg/legal` | 4 | EULA snippets |
| `pkg/supportbundle` | 5 | Operator bundle tar |
| `pkg/proxyupstream` | 5 | Click proxy upstream URLs |
| `pkg/faultproof` | 121 | Fault-test telemetry |
| `pkg/httpresponse` | 117 | Pre-sized admin HTTP errors |
| `pkg/gnetutil` | 4 | gnet listener tuning |
| `pkg/moderatorintel` | 4 | Moderator intel API types |
| `pkg/runtimepaths` | 7 | `var/` path resolution |

**Also moved to `internal/` (not Tier D):** `doctor`, `pgfailover` — operator/shard domains; gated by `pkg_boundary_gate.sh` against reintroduction under `pkg/`.

**Done gates:**

- [x] `bash scripts/ci/pkg_boundary_gate.sh`
- [x] `python3 scripts/dev/sync_package_doc_go.py` (no stale `pkg/doctor`, `pkg/pgfailover` entries)

---

### `remove_pkg_duplicate_servers`

**Priority:** P0

**Gap:** `pkg/broker/server/` and `internal/broker/server/` both exist (~38 files each). Imports already point at `internal/`; pkg copy is dead weight.

**Target:**

- Delete `pkg/broker/server/` entirely
- Delete leftover `pkg/regionproxy/server/` if any files remain after `internal/regionproxy/server/` move
- `pkg/broker/` retains: `client/`, `consumer/`, `log/`, `protocol/`, `doc.go`, `consumer_offset.go`

**Done gates:**

- [x] `test ! -d pkg/broker/server`
- [x] `go build -o /dev/null ./cmd/broker/ ./cmd/region-proxy/`
- [x] `scripts/test/broker_fault_lab.sh` paths use `./internal/broker/server/...` only

---

### `composition_root_inventory`

**Priority:** P0

**Gap:** No single manifest of what must leave `controlplane`. `domains.go` lists prefixes but drifts from extracted `*admin` packages.

**Target file:** `internal/controlplane/domains.go` (update) + this backlog slug table as source of truth.

**Action:** For each `service_*.go` / `team_*.go` / `publisher_*.go`, add row: destination package, bridge file, delete-from-controlplane in same PR as drain.

**Done gates:**

- [x] Every `internal/controlplane/service_*.go` (prod) mapped to a slug below or marked `keep: shell`
- [x] `domains.go` `ManagementDomains` IDs align with `*admin` package names

---

## P1 — flat infra packages

### `flatten_broker_package`

**Priority:** P1 (first)

**Gap:** `internal/broker/server/` redundant subdir; `package server` forced import alias noise.

**Target:** `internal/broker/` with `package broker`. `pkg/broker/` unchanged (client, protocol, log, consumer).

**Done gates:**

- [x] `test ! -d internal/broker/server`
- [x] `go build -o /dev/null ./cmd/broker/ ./internal/broker/...`
- [x] No import paths containing `internal/broker/server`

---

### `flatten_regionproxy_package`

**Priority:** P1 (second)

**Gap:** `internal/regionproxy/server/` same anti-pattern.

**Target:** `internal/regionproxy/` with `package regionproxy`. Topic constant stays `pkg/regionproxy/ingress`.

**Done gates:**

- [x] `test ! -d internal/regionproxy/server`
- [x] `go build -o /dev/null ./cmd/region-proxy/ ./internal/regionproxy/...`

---

## P1 — domain filename hygiene

### `domain_filename_hygiene`

**Priority:** P1 (after each `drain_*` PR)

**Gap:** Files repeat package name (`fraudadmin_handlers.go`) or legacy `service_<domain>_` in controlplane.

**Target:** Per `naming.mdc` — role-based names inside domain dirs; delete drained `service_*.go`; bridges only in controlplane.

**Done gates:**

- [x] No `internal/<pkg>/<pkg>_*.go` in changed packages
- [x] No new `service_<domain>_*.go` under `internal/controlplane/`
- [x] `composition_drain_inventory.go` row removed when source file deleted

---

### `controlplane_wire_only_shell`

**Priority:** P1

**Gap:** `internal/controlplane` ~211 prod `.go`. Target <= 40.

**Keep in controlplane:**

```
adminapi_wire.go
service.go              # Service struct, pool/redis/ch wiring, lazy domain stores
workers.go
postgres_gate.go
register.go
middleware.go
rbac.go
errors.go
handler.go              # top-level health/meta only
ops_reader_bridge.go
*_bridge.go
domains.go
doc.go
```

**Move out:** all domain HTTP bodies, SQL, report math, fraud/RTB/license/dashboard logic (slugs below).

**Done gates:**

- [ ] `find internal/controlplane -maxdepth 1 -name '*.go' ! -name '*_test.go' | wc -l` <= 40
- [ ] No new `service_<domain>.go` files added; new routes land in domain packages

---

### `drain_fraud_service_to_fraudadmin`

**Priority:** P1

**Source (delete from controlplane after move):**

- `service_fraud_api.go`
- `service_fraud_decisions.go`
- `service_fraud_overrides_api.go`
- `service_fraud_presets.go`
- `fraud_ml_snapshot.go` (if not ops-only)

**Destination:**

```
internal/fraudadmin/
  handlers.go           # existing
  decisions.go          # existing
  overrides.go          # existing
  service_api.go        # NEW: methods from service_fraud_*
  host.go               # extend Host port (split if > 12 methods)
```

**Bridge:** `fraudadmin_bridge.go` only maps `Service` -> `fraudadmin.Host`.

**Done gates:**

- [x] `test ! -f internal/controlplane/service_fraud_api.go`
- [x] `go test ./internal/fraudadmin/... -short -count=1`
- [x] OpenAPI routes unchanged; handler package import path updated in `adminapi_wire.go`

---

### `drain_rtb_service_to_rtbadmin`

**Priority:** P1

**Source:**

- `service_rtb.go`
- `service_bid_floor.go`
- RTB handlers still in controlplane (`rtb_*` prefixes per `domains.go`)

**Destination:**

```
internal/rtbadmin/
  handlers.go
  floors_handlers.go    # existing
  service.go          # existing; absorb service_rtb logic
  reconcile.go        # optional: from rtb_reconcile_bridge body
```

**Bridge:** `rtbadmin_bridge.go`, `rtb_reconcile_bridge.go` (shrink).

**Done gates:**

- [x] `test ! -f internal/controlplane/service_rtb.go`
- [x] `test ! -f internal/controlplane/service_bid_floor.go`
- [x] `go test ./internal/rtbadmin/... -short -count=1`

---

### `drain_license_service_to_licensingadmin`

**Priority:** P1

**Source:**

- `service_license.go`
- `service_license_enforce.go`
- `service_eula.go`
- `worker_license_revoke_queue.go`

**Destination:**

```
internal/licensingadmin/
  handlers.go           # existing
  service.go            # existing
  gate.go               # existing
  worker.go             # existing; absorb revoke queue
  eula_handlers.go      # NEW
```

**Bridge:** `licensing_bridge.go`.

**Done gates:**

- [x] License apply/status routes served from `licensingadmin` handlers only
- [x] `go test ./internal/licensingadmin/... -short -count=1`

---

### `drain_dashboards_to_dashboardadmin`

**Priority:** P1

**Source:**

- `service_role_dashboards.go`
- `publisher_dashboard.go`
- `service_campaign_dashboard.go`

**Destination:**

```
internal/dashboardadmin/
  handlers.go
  service_role.go
  service_campaign.go
  publisher.go          # existing
```

**Bridge:** `dashboardadmin_bridge.go`.

**Done gates:**

- [x] Role dashboard DTOs defined only in `dashboardadmin/types.go` / `handlers.go`
- [x] Publisher HTTP routes in `dashboardadmin/publisher_handlers.go`
- [x] `go test ./internal/dashboardadmin/... -short -count=1`
- [ ] No `ghost_*` analytics names in new code (`naming.mdc` / `ui.mdc`)

---

### `drain_platform_team_to_platformadmin`

**Priority:** P1

**Source:**

- `service_platform.go`
- `service_customers.go`
- `team_handlers.go`
- `team_members_handlers.go`
- `team_governance.go`
- `team_budget_approval.go`
- `domain_health.go` (if not already in platformadmin)

**Destination:**

```
internal/platformadmin/
  handlers.go
  customers_handlers.go
  team_handlers.go
  team_members_handlers.go
  governance.go
  budget_approval.go
  domain_health_handlers.go
  store.go
```

**Bridge:** `platform_bridge.go`, `platform_governance_bridge.go`.

**Done gates:**

- [x] No duplicate team HTTP handlers in controlplane
- [x] Customer list/get in `platformadmin/customers.go`
- [x] Domain health service in `platformadmin/domain_health.go`
- [x] `go test ./internal/platformadmin/... -short -count=1`

---

### `drain_billing_service_to_billingadmin`

**Priority:** P1

**Source:**

- `service_workspace_billing.go`
- `service_crypto_billing.go`
- `service_buyer_portfolio.go`
- `handler_billing*` / `billing_*` prefixes still in controlplane

**Destination:**

```
internal/billingadmin/
  handlers.go
  workspace_handlers.go
  workspace_export.go
  crypto_webhook_handlers.go
  invoices.go
  ledger_lines.go
```

**Bridge:** extend `billingadmin` registration in `adminapi_wire.go` (no new bridge if `Host` already wired).

**Done gates:**

- [x] Workspace billing in `billingadmin/workspace_billing.go`
- [x] Crypto webhook processor in `billingadmin/crypto_billing_host.go`
- [x] Buyer portfolio in `dashboardadmin/portfolio.go`
- [x] Billing mutations use `pkg/coldpath` body limits (handlers in `billingadmin`)
- [x] `go test ./internal/billingadmin/... -short -count=1`

---

### `outbox_domain_final_cut`

**Priority:** P1

**Gap:** Outbox worker/handlers live in `internal/outbox/` but controlplane may still hold `outbox_*.go` register paths.

**Target:**

```
internal/outbox/
  worker.go
  register.go
  handlers.go
  host.go
  payloads.go
  ...
```

**Controlplane keeps:** `outbox_bridge.go` only.

**Done gates:**

- [x] No `outbox_*.go` handler bodies in `internal/controlplane/` except bridge
- [x] `go test ./internal/outbox/... -short -count=1`

---

## P2 — hot path split (lead-intent `pipeline` / `filter` / `sink`)

Migrate **one phase per PR**. Order minimizes import cycles: **track** -> **filter** -> **stream** -> **ingest** -> delete monolith glue.

### `ingestion_split_phase_track`

**Priority:** P2

**Move (~18 files):** `track_*.go`, `click_*.go`, `landing_*.go`, `telegram_*` track paths, `handler_track*.go`.

**Target:**

```
internal/track/
  doc.go
  track_core.go
  click_proxy.go
  landing_*.go
  ...
```

**Depends on:** P0 complete.

**Done gates:**

- [ ] `internal/ingestion` prod file count drops by moved files
- [ ] `make test-alloc-gate` on touched hot files (operator tier)
- [ ] `go build -o /dev/null ./cmd/tracker/`

---

### `ingestion_split_phase_filter`

**Priority:** P2

**Move (~23+ files):** `filter*.go`, `filters.go`, `unified_*.go`, `lua_*.go`, `registry_*.go` (filter registry), `fraud_*filter*.go` (not ML scorer).

**Target:**

```
internal/filter/
  doc.go
  engine.go
  unified_filter.go
  ...
```

**Done gates:**

- [ ] `internal/ingestion` does not import `internal/fraud` scoring
- [ ] Holdouts: `TestStreamProducerAdmissionRaceWithoutReserve`, `TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix` still pass under `./internal/filter/...` or `./internal/ingestion/...`

---

### `ingestion_split_phase_stream`

**Priority:** P2

**Move (~12+ files):** `broker_producer*.go`, `stream_producer*.go`, admission, rollback, `clickhouse_store*.go` (if sink-bound).

**Target:**

```
internal/stream/
  doc.go
  producer.go
  broker_producer.go
  clickhouse_store.go   # or internal/sink/ if CH is shared with processor
```

**Done gates:**

- [ ] Broker-primary wiring tests still in `test-fault` tier (`boundaries.mdc`)
- [ ] No dual XADD regression (holdout cited in PR)

---

### `ingestion_split_phase_ingest`

**Priority:** P2

**Move:** gnet `handler.go`, `conn_*.go`, `parser/`, `http*_*.go`, ingress chaos paths.

**Target:**

```
internal/ingest/
  doc.go
  handler.go
  conn_gnet.go
  parser/
```

**Final:** `internal/ingestion/` becomes thin re-export shim **one release**, then deleted.

**Done gates:**

- [ ] `TestChaos_CrossHop_NginxGnet` `differential_count=0`
- [ ] Tracker p99 budget unchanged in load-test tier (not microbench)

---

### `campaign_subpackages_runtime_worker`

**Priority:** P2

**Gap:** `internal/campaign` ~78 prod `.go` (above ceiling).

**Target:**

```
internal/campaign/
  doc.go
  handlers.go
  dto.go
  runtime.go            # CRUD, publish, patch
  editor_handlers.go
  wizard_store.go
  worker/
    doc.go
    delivery.go
    pacing.go
    autoscale.go
```

**Done gates:**

- [ ] Root `campaign/` <= 40 prod `.go`; `campaign/worker/` <= 40
- [ ] `campaign -X-> controlplane` import cycle preserved

---

### `reports_fraud_subtree_stable`

**Priority:** P2

**Gap:** Customer fraud reports oscillate between `internal/reports/fraud/` and flat `reports/`; compile errors block `cmd/control` build.

**Target:**

```
internal/reports/
  doc.go
  handlers.go
  catalog.go
  fraud/
    doc.go
    customer_fraud_overview.go
    customer_fraud_by_type.go
    customer_fraud_by_dimension.go
    customer_fraud_evidence.go
    fraud_compat.go       # re-exports for controlplane aliases only
```

**Done gates:**

- [ ] `go build -o /dev/null ./internal/reports/...`
- [ ] `RegisterFraudReportRoutes` defined once in `reports/fraud/routes.go` or `reports/handlers.go`
- [ ] `go test ./internal/reports/... -short -run Fraud -count=1`

---

## P3 — secondary god packages

### `licensing_package_split`

**Priority:** P3

**Target:**

```
internal/licensing/
  verify/       # JWT crypto, HWID
  entitlements/ # SKU gates, snapshot
  trial/        # trial registry hooks
```

Keep public API via type aliases in `licensing/doc.go` root if needed for one release.

---

### `domain_package_split`

**Priority:** P3

**Target:**

```
internal/domain/
  event.go          # hot types stay import-light
  budget/           # AssertBudgetInvariant, spend types
  shard/            # slot map, static shard
```

Hot path imports only `domain` event + budget packages, not PG types.

---

### `cmd_deps_pattern`

**Priority:** P3

**Gap:** Lead-intent `cmd/parser` delegates to `internal/app/buildDeps`. Several ad-event `main.go` files are long.

**Target:** Each `cmd/<binary>/main.go` <= 150 lines; wire in `cmd/<binary>/wire.go` or `internal/control/runner.go`.

| Binary | Wire file |
| :--- | :--- |
| `tracker` | `cmd/tracker/wire.go` |
| `control` | `internal/control/deps.go` (existing pattern) |
| `broker` | `cmd/broker/serve.go` + `wire.go` |

**Done gates:**

- [ ] No SQL or handler logic in any `cmd/*/main.go`

---

## P4 — enforcement

### `structure_ci_gate_package_size`

**Priority:** P4

**Gap:** Ceilings exist in `.mdc` but are not CI-enforced.

**Target script:** `scripts/ci/package_size_gate.sh`

- Fail if any `internal/*` or `pkg/*` root package has > 40 prod `.go` files (allowlist file for grandfathered until slug closed)
- Fail if `pkg/**/*.go` (non-test) imports `internal/`

**Done gates:**

- [ ] Gate wired in `pr_fast.sh` after P1+P2 shrink below allowlist
- [ ] Allowlist empty or only `ingestion` until `ingestion_split_phase_ingest` ships

---

## What we deliberately do not copy from lead-intent

| lead-intent pattern | ad-event choice |
| :--- | :--- |
| Single `cmd/parser` | Keep multi-binary (`tracker`, `processor`, `broker`, …) |
| No bridges | Keep `*_bridge.go` in controlplane; shrink them |
| `internal/sources/<name>` plugins | Use existing `internal/rtb`, `migrationsource`, `ingestion` parsers — only split when > 40 files |
| Mongo-only sink | Keep CH/PG/Redis/broker sinks in `stream/` + `processor` |
| `pkg/bpfenv` only | Allow richer `pkg/` util set; forbid `internal/*` imports |

---

## PR sizing guide

| PR type | Max scope |
| :--- | :--- |
| P0 slug | 1 slug, all done gates |
| P1 drain | 1 admin domain, 1 bridge shrink, delete matching `service_*.go` |
| P2 ingestion phase | 1 of `track`/`filter`/`stream`/`ingest` |
| P3 split | 1 package subfolder |

Do not combine P1 drain + P2 ingestion split in one PR (review blast radius).
