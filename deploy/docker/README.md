# Docker images

Dockerfiles live at the repo root and under `deploy/`. Default compose builds use the main `Dockerfile` with per-service `entrypoint` overrides.

## Dockerfile matrix

| Dockerfile | Image role | Binaries / output | Base | Compose usage |
| :--- | :--- | :--- | :--- | :--- |
| `Dockerfile` | Core platform services | tracker, processor, management, auth, payment, billing, notifier, ivt-detector, fraud-scorer, broker, region-proxy, log-shipper, control, alertmanager-telegram | distroless static | Default `docker-compose.yaml` (`build.dockerfile: Dockerfile`) |
| `Dockerfile.log-compactor` | Log compaction worker | log-compactor | distroless static | `tools` profile |
| `Dockerfile.log-evacuator` | Tracker log archive | log-evacuator | distroless static | `tools` profile |
| `deploy/edge/xdp/Dockerfile` | Edge XDP filter + BPF sync | edge-xdp, edge-bpf-sync | debian bookworm-slim | Manual / host ingress (not default compose) |
| `deploy/ml/Dockerfile` | Fraud model bootstrap CronJob | `ml/train.py` | python 3.12-slim | k8s / manual |

### Main image (`Dockerfile`)

Multi-stage: `golang:1.25-alpine` builder, `gcr.io/distroless/static-debian12` runtime. `ENTRYPOINT` defaults to `/tracker`; compose sets `entrypoint` per service.

```bash
docker build -t espx-tracker .
docker build -t espx-management --build-arg ... # same Dockerfile; override entrypoint at run time
```

### Utility images

```bash
docker build -f Dockerfile.log-compactor -t espx-log-compactor .
docker build -f Dockerfile.log-evacuator -t espx-log-evacuator .
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
