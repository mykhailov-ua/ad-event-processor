# GitHub Actions workflows

Describe each workflow file under this directory. Run merge gates locally with the scripts named in [docs/DEVELOPMENT.md](../../docs/DEVELOPMENT.md).

**Repository variable:** `PERF_RUNNER_LABEL` — when set, perf/BPF workflows schedule jobs on that self-hosted label and run strict probes. When empty, those jobs use `ubuntu-latest` and mostly smoke or skip real BPF/perf work.

**Release secrets:** `GARBLE_SEED_SALT`, `ASSET_SEAL_SALT` — required for garbled release builds in `release-images.yaml`.

---

## `ci.yaml`

**Trigger:** push to `main`; pull requests to `main`; `workflow_dispatch`.

**Purpose:** Primary merge gate.

**Jobs (path-filtered):** `merge-lint` (Go golangci + vet + gopls, Lua luacheck/LuaLS, shellcheck, Python ruff, buf lint, actionlint, TS when `web/` exists, compose/nginx/OpenAPI) → `merge-pr-fast` → parallel `race-short`, `integration`, `govulncheck`, `perf-smoke`; conditional OpenRTB fuzz and fraud-model tests; on `main` only — full test tier, resilience (`scripts/fault/run.sh`), license red-team.

**Runner:** `ubuntu-latest`. Go 1.25.12. Node 22 for pr-fast. Python 3.12 for fraud-model job.

**Gate:** Required for PR merge to `main`.

---

## `perf-gate.yaml`

**Trigger:** push to `main` when hot-path paths change; `workflow_dispatch`.

**Purpose:** Allocation and perf regression on ingest/rtb-related changes.

**Job:** `scripts/test/load/gate_run.sh`. Strict mode when `PERF_RUNNER_LABEL` is set (CPU stabilize, benchstat artifacts); otherwise zero-alloc smoke only.

**Gate:** Main-branch hot-path changes; not on every PR path.

---

## `perf-nightly.yaml`

**Trigger:** Monday 03:00 UTC; `workflow_dispatch`.

**Purpose:** Nightly escape-heap, Redis Lua bench, broker protocol bench, cache-miss A/B (continue-on-error). Compares against cached baselines.

**Gate:** Regression signal only; not a PR blocker.

---

## `bpf-resource-gate.yaml`

**Trigger:** PR and `main` push on ingest/controlplane/loadreport/BPF script paths; `workflow_dispatch` with optional export soak input.

**Purpose:** Short BPF resource smoke (`scripts/ci/bpf/resource.sh`, 90s, strict when label set). Optional export soak via dispatch.

**Runner:** Only runs when `PERF_RUNNER_LABEL` is non-empty.

**Gate:** Path-scoped when self-hosted runner is configured.

---

## `bpf-nightly.yaml`

**Trigger:** Tuesday 04:00 UTC; `workflow_dispatch`.

**Purpose:** Three nightly BPF jobs — hot baseline, cold soak, report-export soak (`scripts/test/bpf/nightly_job.sh`). Caches baselines under `.ci-baselines/bpf/`.

**Gate:** Nightly regression; not PR merge.

---

## `parser-fuzz-nightly.yaml`

**Trigger:** Sunday 05:00 UTC; `workflow_dispatch`.

**Purpose:** Long fuzz runs on ingest parsers (`ParseTrackJSON`, `SkipJSONValueBudget`, `HTTP1Chunked`, `ParseOpenRTB3FSM`) — 2h each. Uploads corpus on failure.

**Timeout:** 600 minutes. Uses `PERF_RUNNER_LABEL` when set.

---

## `license-fuzz-nightly.yaml`

**Trigger:** Monday 04:00 UTC; `workflow_dispatch`.

**Purpose:** Fuzz `internal/licensing/` JWT decode paths (10m + 5m + 5m). Corpus artifact on failure.

---

## `compose-fault-nightly.yaml`

**Trigger:** Monday 04:00 UTC; `workflow_dispatch`.

**Purpose:** Host tune + `scripts/fault/compose_fault_drill.sh all`. Uploads fault and RAM proof logs.

**Runner:** `PERF_RUNNER_LABEL` or `ubuntu-latest` (drill depth depends on host).

---

## `enterprise-resilience.yaml`

**Trigger:** `workflow_dispatch` only.

**Purpose:** Manual multi-region drill (`scripts/test/multi_region_resilience_drill.sh`) and XDP drill (clang/llvm/libbpf + `scripts/test/edge/xdp_resilience_drill.sh`).

**Gate:** Operator drill; not scheduled.

---

## `sentinel-resilience.yaml`

**Trigger:** push to `main`; `workflow_dispatch`.

**Purpose:** Redis Sentinel failover smoke — `scripts/fault/sentinel_failover_env.sh` then `scripts/test/sentinel.sh`.

**Gate:** Main push only (not PR in this file).

---

## `admin-stack-e2e.yaml`

**Trigger:** Monday 03:00 UTC; `workflow_dispatch`.

**Purpose:** Admin stack E2E via `scripts/test/admin_stack_e2e.sh` when `web/scripts/build.mjs` exists.

**Behavior:** Exits successfully when `web/` tree is absent (current default).

---

## `release-images.yaml`

**Trigger:** push tags `v*`; GitHub `release` published; `workflow_dispatch` with optional tag input.

**Purpose:** Build and push `pilot` and `pilot-ingest` images to GHCR; garble and asset-seal salts; binary surface check; `make release-installer`.

**Secrets:** `GARBLE_SEED_SALT`, `ASSET_SEAL_SALT` (required). `GITHUB_TOKEN` for registry push.

**Gate:** Release path only.
