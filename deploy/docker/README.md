# Docker images

Dockerfiles live under `deploy/docker/` and `deploy/{edge,ml}/`. Default compose builds use `deploy/docker/Dockerfile` with per-service `entrypoint` overrides.

## Dockerfile matrix

| Dockerfile | Image role | Binaries / output | Base | Compose usage |
| :--- | :--- | :--- | :--- | :--- |
| `deploy/docker/Dockerfile` | Core platform services | tracker, processor, management, auth, payment, billing, notifier, ivt-detector, fraud-scorer, broker, region-proxy, log-shipper, control, alertmanager-telegram | distroless static | `deploy/compose/docker-compose.yaml` |
| `deploy/docker/Dockerfile.log-compactor` | Log compaction worker | log-compactor | distroless static | `tools` profile |
| `deploy/docker/Dockerfile.log-evacuator` | Tracker log archive | log-evacuator | distroless static | `tools` profile |
| `deploy/edge/xdp/Dockerfile` | Edge XDP filter + BPF sync | edge-xdp, edge-bpf-sync | debian bookworm-slim | Manual / host ingress (not default compose) |
| `deploy/ml/Dockerfile` | Fraud model bootstrap CronJob | `artifact_bootstrap.py fit-validate` | python 3.12-slim | k8s / manual |

### Main image (`deploy/docker/Dockerfile`)

Multi-stage: `golang:1.25-alpine` builder, `gcr.io/distroless/static-debian12` runtime. `ENTRYPOINT` defaults to `/tracker`; compose sets `entrypoint` per service.

```bash
docker build -f deploy/docker/Dockerfile -t espx-tracker .
docker build -f deploy/docker/Dockerfile -t espx-management .  # override entrypoint at run time
```

### Utility images

```bash
docker build -f deploy/docker/Dockerfile.log-compactor -t espx-log-compactor .
docker build -f deploy/docker/Dockerfile.log-evacuator -t espx-log-evacuator .
docker build -f deploy/ml/Dockerfile -t espx-ml-bootstrap .
```

## edge-xdp manual build

Kernel XDP filter and Redis map sync run outside the default stack. Requires privileged host networking.

### Container image

From repo root:

```bash
docker build -f deploy/edge/xdp/Dockerfile -t espx-edge-xdp .
```

### BPF object (host / CI)

```bash
make -C deploy/edge/xdp
# output: deploy/edge/xdp/bpf/edge_filter.o
```

Requires `clang` and BPF headers. Source: `deploy/edge/xdp/bpf/edge_filter.c` (also referenced from `internal/edge/bpf`).

### Run

```bash
docker run --rm -it --privileged --network host \
  -e INGRESS_INTERFACE=eth0 \
  -e BROKER_REDIS_URL=redis://127.0.0.1:6379/0 \
  espx-edge-xdp
```

`INGRESS_INTERFACE` is required. See `deploy/edge/README.md` for Phase 0 host tuning before enabling XDP.
