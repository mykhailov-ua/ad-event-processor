# Self-hosted runner: BPF resource gate

Required for **Gate · bpf-resource** (`scripts/ci/bpf_resource_gate.sh`) when `PERF_RUNNER_LABEL` is set on the repository.

## Hardware and OS

| Requirement | Notes |
| :--- | :--- |
| Linux amd64 | Same arch as production tracker images |
| Kernel 5.8+ | tracepoint + uprobe attach |
| BTF | `/sys/kernel/btf/vmlinux` must exist (CO-RE probe load) |
| 8+ GiB RAM | compose smoke stack + loadgen |
| 4+ CPU cores | constrained compose profile |

## Privileges

`bpf-collector` calls `rlimit.RemoveMemlock()` and attaches kernel probes. One of:

- Run the GitHub Actions runner service as **root**, or
- Passwordless sudo for the runner user (`sudo -n true`), or
- `ESPX_BPF_SUDO_PASS` in runner environment (not recommended for shared CI)

Verify locally:

```bash
sudo bash scripts/test/bpf_requirements.sh
sudo bash scripts/dev/bpf_session.sh start
sudo bash scripts/dev/bpf_session.sh stop
```

## Software

```bash
# Ubuntu 24.04 example
sudo apt-get install -y clang llvm libbpf-dev bpftool docker.io docker-compose-plugin
```

- Go **1.25.12** (match `go.mod` / workflows)
- Docker with access for the runner user (`docker ps` without sudo)

## GitHub configuration

1. Register self-hosted runner with label matching repo variable **`PERF_RUNNER_LABEL`** (e.g. `bidshard-perf`).
2. Settings → Secrets and variables → Actions → Variables → `PERF_RUNNER_LABEL` = runner label.
3. Optional: mark **Gate · bpf-resource** as required in branch protection (after first green run).

## CPU governor

Gate script calls `scripts/perf/stabilize_cpu.sh` when strict. On bare metal:

```bash
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

## Local reproduction

```bash
export BPF_GATE_STRICT=true
export ESPX_BPF_PROBE=1
bash scripts/ci/bpf_resource_gate.sh
```

Evaluate an existing session only:

```bash
BPF_GATE_STRICT=1 go run ./cmd/load-report bpf-gate var/load-test/<session> --prom http://127.0.0.1:9190
BPF_GATE_STRICT=1 BPF_COLD_GATE=1 go run ./cmd/load-report bpf-gate var/load-test/<session>
go run ./cmd/load-report bpf-gate-compare .ci-baselines/bpf/hot var/load-test/<session>
```

Optional OTel export for ringbuf slow events:

```bash
export ESPX_BPF_OTEL_ENDPOINT=http://127.0.0.1:4318
# or OTEL_EXPORTER_OTLP_ENDPOINT / OTEL_EXPORTER_OTLP_LOGS_ENDPOINT
```

## Failure artifacts

| File | Content |
| :--- | :--- |
| `bpf-gate.md` | per-check PASS/FAIL/SKIP table |
| `bpf/maps/summary.json` | aggregated probe stats |
| `bpf/events.ndjson` | slow syscall/uprobe events |
| OTLP `/v1/logs` | optional ringbuf export when `ESPX_BPF_OTEL_ENDPOINT` is set |
| `bpf_resource_gate.log` | orchestrator log |

## Security

- Do not expose runner Docker socket to untrusted PR forks if workflow runs on `pull_request` from forks (use `pull_request_target` only with review policy, or restrict to same-repo PRs).
- BPF attach is equivalent to root on the host; isolate runner VM from production networks.
