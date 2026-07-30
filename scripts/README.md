# scripts/

One folder level only. Go CLIs live in `cmd/`. Prefer **`Makefile`** targets when they exist.

```bash
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
```

`lib/paths.sh` exports `ROOT`, `SCRIPTS`, and loads `lib/safe_paths.sh` (`safe_rm_rf`, `safe_validate_codegen_configs`, …).

## Layout

| Folder | Purpose | Entry command |
| :--- | :--- | :--- |
| `lib/` | Shared `ROOT`/`SCRIPTS` helpers | `source scripts/lib/paths.sh` |
| `ci/` | Merge gates, codegen, layout checks | `bash scripts/ci/local_check.sh` |
| `dev/` | Local compose stack and preflight | `bash scripts/dev/stack.sh full` |
| `local-dev/` | Stable aliases for docs/CI smoke | `bash scripts/local-dev/dev_preflight.sh` |
| `load/` | Loadgen, BPF sessions, drills | `make load-test-bpf` |
| `perf/` | Benchmark gates and nightly jobs | `bash scripts/perf/gate_run.sh` |
| `perf-gate/` | Stable alias for perf smoke | `PERF_GATE_STRICT=false bash scripts/perf-gate/perf_gate_run.sh` |
| `fault/` | Fault injection and resilience proofs | `make test-resilience` |
| `edge/` | Ingress sysctl/NIC tuning and rollout | `bash scripts/edge/phase0.sh` |
| `edge-tuning/` | Stable alias for edge Phase 0 | `bash scripts/edge-tuning/edge_phase0.sh` |
| `redis/` | Shard topology and campaign migration wrappers | `bash scripts/redis/verify_topology.sh .env` |
| `deploy/` | k8s, cold/hot path, post-deploy reconcile | `bash scripts/deploy/hot_path_up.sh` |
| `test/` | Script self-tests (not CI entrypoints) | `bash scripts/test/parse_dfa_test.sh` |

## Makefile entrypoints

| Target | Script |
| :--- | :--- |
| `make gen` | `ci/gen.sh` |
| `make check-local` | `ci/local_check.sh` |
| `make test-full` | `ci/full_test.sh` |
| `make test-resilience` | `fault/run.sh` |
| `make test-broker-fault-lab` | `fault/broker_fault_lab.sh` |
| `make test-sentinel-resilience` | `fault/sentinel.sh` |
| `make management-domain-coverage` | `ci/management_domain_coverage.sh` |
| `make bpf-dev` | `dev/bpf_setup.sh` |
| `make load-test-bpf` | `load/malformed.sh business` |
| `make openapi-lint` | `ci/openapi.sh` |
| `make check-scripts-layout` | `ci/check_scripts_layout.sh` |
| `make dev-preflight-smoke` | `local-dev/dev_preflight.sh` |
| `make perf-gate-smoke` | `perf-gate/perf_gate_run.sh` |
| `make edge-phase0` | `edge-tuning/edge_phase0.sh` |

## P09 smoke (clean clone, no stack)

These exit 0 without a running compose stack:

```bash
bash scripts/local-dev/dev_preflight.sh          # configs only (PREFLIGHT_SMOKE=1)
PERF_GATE_STRICT=false bash scripts/perf-gate/perf_gate_run.sh
bash scripts/edge-tuning/edge_phase0.sh          # non-STRICT: warns on sysctl/NIC/nginx
```

Full dependency check (requires PG/Redis/CH):

```bash
PREFLIGHT_SMOKE=0 bash scripts/dev/preflight.sh
bash scripts/ci/deps.sh
bash scripts/dev/smoke_local.sh
```

## Dangerous (read before run)

| Script | Risk |
| :--- | :--- |
| `load/host_tune.sh` | host sysctl / IRQ — root |
| `edge/sysctl.sh apply`, `edge/nic_tune.sh apply` | kernel / NIC — root |
| `perf/stabilize_cpu.sh` | CPU governor — whole machine |
| `ci/prepare_test.sh`, `load/prepare_constrained_stack.sh` | TRUNCATE PG/CH |
| `deploy/migrate_campaign.sh`, `redis/migrate_campaign.sh` | Redis DUMP/RESTORE |
| `fault/sentinel.sh` | `docker kill` Redis |
| `deploy/install_k3s.sh` | installs k3s |
| `load/malformed.sh`, `load/spike.sh` | high RPS; BPF needs root |
