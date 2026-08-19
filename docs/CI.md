# CI

GitHub Actions: `.github/workflows/`. Dependabot: `.github/dependabot.yml`. Job prefix: `Gate · …`.

Run locally before PR: `bash scripts/ci/pr_fast.sh` (= **Gate · merge-pr-fast**).

## Pipeline Overview

```text
PR / push main
  ci.yaml ── merge-pr-fast ──┬── merge-race-short
                             ├── merge-integration
                             ├── merge-govulncheck (optional)
                             ├── merge-perf-smoke (optional)
                             ├── merge-openrtb-fuzz (path-filter)
                             └── merge-fraud-model (path-filter)

push main only
  ├── main-full-test, main-resilience, main-license-red-team

scheduled / manual
  perf-gate, bpf-resource-gate, sentinel-resilience, perf-nightly, bpf-nightly
  parser-fuzz-nightly, license-fuzz-nightly, compose-fault-nightly
  admin-stack-e2e, enterprise-resilience

tag v* → release-images.yaml
```

Overlapping runs cancel in-progress (`cancel-in-progress: true`), except nightly/release.

## Workflows

| Workflow | Trigger | Runner | Purpose |
| :--- | :--- | :--- | :--- |
| [ci.yaml](../.github/workflows/ci.yaml) | PR, push `main` | `ubuntu-latest` | Merge gates |
| [perf-gate.yaml](../.github/workflows/perf-gate.yaml) | Push `main` (hot paths) | `PERF_RUNNER_LABEL` or hosted | Strict perf |
| [bpf-resource-gate.yaml](../.github/workflows/bpf-resource-gate.yaml) | PR/push BPF paths | Self-hosted only | eBPF load gate |
| [sentinel-resilience.yaml](../.github/workflows/sentinel-resilience.yaml) | Push `main` | hosted | Sentinel failover |
| [perf-nightly.yaml](../.github/workflows/perf-nightly.yaml) | Mon 03:00 UTC | perf runner | Escape, Lua, broker benches |
| [bpf-nightly.yaml](../.github/workflows/bpf-nightly.yaml) | Tue 04:00 UTC | perf runner | Hot + cold soak |
| [parser-fuzz-nightly.yaml](../.github/workflows/parser-fuzz-nightly.yaml) | Sun 05:00 UTC | perf runner | Parser fuzz 2h/target |
| [license-fuzz-nightly.yaml](../.github/workflows/license-fuzz-nightly.yaml) | Mon 04:00 UTC | hosted | JWT/HWID fuzz |
| [compose-fault-nightly.yaml](../.github/workflows/compose-fault-nightly.yaml) | Mon 04:00 UTC | perf runner | Compose fault drill |
| [admin-stack-e2e.yaml](../.github/workflows/admin-stack-e2e.yaml) | Mon 03:00 UTC | hosted | Admin Playwright + manual memory smoke (paginate campaigns, ops tabs, theme toggle; heap must not grow monotonically) |
| [enterprise-resilience.yaml](../.github/workflows/enterprise-resilience.yaml) | Manual | hosted | MR + XDP drills |
| [release-images.yaml](../.github/workflows/release-images.yaml) | Tag `v*` | hosted | GHCR + installer |

Toolchain: Go **1.25.12**, Node **22**.

## `ci.yaml` Merge Gates

### Path filter (`dorny/paths-filter@v3`)

| Output | Paths |
| :--- | :--- |
| `openrtb` | `internal/openrtb/**`, `internal/ingestion/openrtb*.go`, `scripts/test/openrtb_fuzz_smoke.sh` |
| `model` | `model/**` |
| `go` | `**/*.go`, `go.mod`, `go.sum`, `api/**`, `internal/**/queries/**` |
| `integration` | go deps + `scripts/**`, `tests/**`, `deploy/**`, `docker-compose*.yaml`, `Dockerfile*` |

### Required checks (branch protection)

| Check | Script | Blocks |
| :--- | :--- | :--- |
| **Gate · merge-pr-fast** | `bash scripts/ci/pr_fast.sh` | Yes |
| **Gate · merge-race-short** | `bash scripts/ci/race_short.sh` | Yes |
| **Gate · merge-integration** | `bash scripts/ci/integration_test.sh` | Yes |
| Gate · merge-govulncheck | `bash scripts/ci/govulncheck.sh` | Optional |
| Gate · merge-openrtb-fuzz | `make openrtb-fuzz-smoke` | Optional (path) |
| Gate · merge-fraud-model | `bash scripts/ci/fraudtrain.sh` | Optional (path) |
| Gate · merge-perf-smoke | `bash scripts/ci/perf_smoke.sh` | Optional |

### `merge-pr-fast` steps

Config validation, docs tier A, compliance, ClickHouse direct, `lint_go_gate.sh`, integration slop gates, anti-slop, SQL safety, hot/cold static gates, escape heap gate, `make test-alloc-gate`, `make test-fast`, shard0 nil gate, cold-path JSON gate, CAPI staging, `admin_web.sh`.

### Main-only

| Check | Script | When |
| :--- | :--- | :--- |
| Gate · main-full-test | `bash scripts/ci/full_test.sh` | Push `main` (integration paths) |
| Gate · main-resilience | `bash scripts/fault/run.sh` | Push `main` |
| Gate · main-license-red-team | `make license-red-team` | Push `main` |

`main-resilience`: `RESILIENCE_MIN_PROOFS=52`, `RESILIENCE_MIN_PROOFS_MR=12`.

## Performance Gates

### PR smoke (`merge-perf-smoke`)

`PERF_GATE_STRICT=false`, `GOMAXPROCS=1` → `scripts/ci/perf_smoke.sh`. No benchstat on hosted.

### Main strict (`perf-gate.yaml`)

Hot-path file changes on `main`. Self-hosted: `stabilize_cpu.sh`, benchstat vs `main`. Production claims need `PERF_GATE_STRICT=true bash scripts/test/gate_run.sh` or `make test-alloc-gate`.

## BPF Resource Gate

Requires repo var `PERF_RUNNER_LABEL`. 90s smoke, `BPF_GATE_STRICT=true`, `AD_EVENT_PROCESSOR_BPF_PROBE=1`. Script: `scripts/ci/bpf_resource_gate.sh`.

**Self-hosted runner:** Linux amd64, kernel 5.8+, BTF (`/sys/kernel/btf/vmlinux`), 8+ GiB RAM, 4+ cores. Privileges: runner as root, passwordless sudo, or `AD_EVENT_PROCESSOR_BPF_SUDO_PASS`. Software: Go 1.25.12, Docker, `clang llvm libbpf-dev bpftool`. Label runner with `PERF_RUNNER_LABEL` repo variable.

```bash
sudo bash scripts/test/bpf_requirements.sh
export BPF_GATE_STRICT=true AD_EVENT_PROCESSOR_BPF_PROBE=1
bash scripts/ci/bpf_resource_gate.sh
BPF_GATE_STRICT=1 go run ./cmd/load-report bpf-gate var/load-test/<session> --prom http://127.0.0.1:9190
```

Artifacts: `bpf-gate.md`, `bpf/maps/summary.json`, `bpf/events.ndjson`, `bpf_resource_gate.log`.

### BPF architecture

| Layer | Scope | Blocks merge |
| :--- | :--- | :--- |
| A — static | alloc-gate, race, integration, anti-slop | Yes (PR) |
| B — microbench | `gate_bench.sh`, perf-gate +12% | PR smoke; main strict |
| C — eBPF load | `load-report bpf-gate` | When perf runner set |
| D — cold soak | admin, query budget, goleak | Nightly |
| E — security | parser fuzz, SQL safety | Parallel |

### Hot-path BPF thresholds

| Metric | WARN | FAIL |
| :--- | ---: | ---: |
| `filter_check` uprobe p99 | > 500 µs | > 1 ms |
| `process_track` uprobe p99 | > 2 ms | > 5 ms |
| tracker handler p99 | > 50 ms | **≥ 80 ms** |
| Redis Lua p99/shard | > 5 ms | **≥ 10 ms** |
| `tracker_outbound_connect` | — | **> 0** |
| `tracker_rss_delta_kb` | — | **> 5120** |
| loadgen on-CPU % | > 15% | > 25% |

Static gate: `hot_path_static_gate.sh` — forbid `fmt.Sprintf`, `context.With*`, hot `defer` in ingestion/rtb/tracker.

### Cold-path BPF thresholds

| Metric | FAIL |
| :--- | :--- |
| `fd_delta` after settle | **> 0** |
| `max_rss_delta_kb` (control/processor) | **> 51200** |
| `ch_spool_segments` / `disk_gate_degraded` | **> 0** |
| `broker_disk_writable` | **< 1** |
| `redis_pool_timeouts_rate` | **> 0** |
| `processor_pg_acquire_wait_p99_ms` | **≥ 100** |
| `stream_producer_queue_depth_p99` | **≥ 1000** |
| Admin handler p99 | > 500 ms |

## Nightly Schedule (UTC)

| Workflow | Schedule | Jobs |
| :--- | :--- | :--- |
| perf-nightly | Mon 03:00 | Escape, Lua, broker, cache-miss |
| admin-stack-e2e | Mon 03:00 | `admin_stack_e2e.sh` |
| license-fuzz-nightly | Mon 04:00 | JWT fuzz |
| compose-fault-nightly | Mon 04:00 | `compose_fault_drill.sh all` |
| bpf-nightly | Tue 04:00 | Hot baseline + cold soak |
| parser-fuzz-nightly | Sun 05:00 | 4 fuzz targets × 2h |

Baselines: `.ci-baselines/`.

## Resilience

| Workflow | Script |
| :--- | :--- |
| sentinel-resilience | `scripts/fault/sentinel.sh` |
| enterprise-resilience | `mr_resilience_drill.sh`, `xdp_resilience_drill.sh` |
| main-resilience (ci.yaml) | `scripts/fault/run.sh` |

Runbooks: [DEVELOPMENT.md](DEVELOPMENT.md) §7.

## Release Images

Tag `v*` → GHCR. Profiles: `pilot` (control, processor, tracker), `pilot-ingest` (processor, tracker). Secrets: `GARBLE_SEED_SALT`, `ASSET_SEAL_SALT`. Follow-up: `make release-installer`.

## Self-Hosted Runner

Var **`PERF_RUNNER_LABEL`** must match runner label. Enables: bpf-resource-gate, perf-gate strict, bpf-nightly, stabilized CPU.

## Go Test Tiers

| Target | Command | Scope |
| :--- | :--- | :--- |
| Fast | `make test-fast` | `go test -short` |
| Race short | `bash scripts/ci/race_short.sh` | `-race -short` on `internal/`, `pkg/` |
| Integration | `make test-integration` | Testcontainers; 30m timeout |
| Fault | `make test-fault` | `-run Fault` |
| Alloc gate | `make test-alloc-gate` | Zero-alloc microbenches |

Integration skips under `-short` with reason string; gate: `integration_skip_reason_gate.sh`.

## Lab Script Skip Matrix

Scripts exit 0 with `skip (…)` when preconditions missing.

| Script | Preconditions |
| :--- | :--- |
| `xdp_resilience_drill.sh` | BTF, root/CAP |
| `edge_xdp_compose_smoke.sh` | BTF, Docker |
| `nginx_ktls_smoke.sh` | Docker; live needs `tls` module |
| `broker_primary_smoke.sh` | Docker + broker for live |
| `cpu_isolation_smoke.sh` | `CPU_ISOLATION_ENABLED=1`, running tracker |
| `admin_stack_e2e.sh` | Control :8188, bootstrap env |

## Static Gates

- **Anti-slop:** bare `t.Skip()`, `_ = err`, bench naming, UI slop.
- **SQL safety:** no raw `fmt.Sprintf` SQL; use sqlc.

## Dependencies

Dependabot: Go patch+minor grouped/monthly. Govulncheck optional. Actions pins: manual update.

## Failure Artifacts

`merge-*-log`, `main-*-log`, `perf-gate-*`, `bpf-resource-gate`, `sentinel-log`, `compose-fault-log`, fuzz corpora. Retention 7–14 days.

## Local Commands

```bash
bash scripts/ci/pr_fast.sh              # merge-pr-fast
bash scripts/ci/race_short.sh           # merge-race-short
bash scripts/ci/integration_test.sh     # merge-integration
bash scripts/ci/full_test.sh            # main-full-test
bash scripts/fault/run.sh               # main-resilience
bash scripts/ci/perf_smoke.sh           # merge-perf-smoke
PERF_GATE_STRICT=true bash scripts/test/gate_run.sh
make check-local
bash scripts/fault/sentinel.sh
export BPF_GATE_STRICT=true AD_EVENT_PROCESSOR_BPF_PROBE=1
bash scripts/ci/bpf_resource_gate.sh
```
