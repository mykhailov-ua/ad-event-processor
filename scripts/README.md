# scripts/

```
scripts/
  lib/          paths.sh, safe_paths.sh
  ci/           merge gates, codegen
  dev/          local stack
  load/         loadgen + BPF + go run load-report
  perf/         benchmark gates
  fault/        fault injection CI
  edge/         ingress tuning & rollout
  deploy/       k8s, redis, fraud train
  test/         script self-tests (not CI entrypoints)
```

One folder level only. Go CLIs live in `cmd/`. Use **`Makefile`** targets when possible.

```bash
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
```

## Entrypoints

| Target | Script |
|--------|--------|
| `make gen` | `ci/gen.sh` |
| `make check-local` | `ci/local_check.sh` |
| `make test-full` | `ci/full_test.sh` |
| `make test-resilience` | `fault/run.sh` |
| `make test-broker-fault-lab` | `fault/broker_fault_lab.sh` |
| `make test-sentinel-resilience` | `fault/sentinel.sh` |
| `make bpf-dev` | `dev/bpf_setup.sh` |
| `make load-test-bpf` | `load/malformed.sh business` |
| `make openapi-lint` | `ci/openapi.sh` |

## Dangerous (read before run)

| Script | Risk |
|--------|------|
| `load/host_tune.sh` | host sysctl / IRQ — root |
| `edge/sysctl.sh apply`, `edge/nic_tune.sh apply` | kernel / NIC — root |
| `perf/stabilize_cpu.sh` | CPU governor — whole machine |
| `ci/prepare_test.sh`, `load/prepare_constrained_stack.sh` | TRUNCATE PG/CH |
| `deploy/migrate_campaign.sh` | Redis DUMP/RESTORE |
| `fault/sentinel.sh` | `docker kill` Redis |
| `deploy/install_k3s.sh` | installs k3s |
| `load/malformed.sh`, `load/spike.sh` | high RPS; BPF needs root |
