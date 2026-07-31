# Broker HA lab

Two-node mmap log broker with Redis coordination and HAProxy frontends. Host networking only; intended for local failover drills and `make test-broker-fault-lab`.

## Build binary

```bash
mkdir -p deploy/broker/bin
CGO_ENABLED=0 go build -o deploy/broker/bin/espx-broker ./cmd/broker
```

Override the mount path with `ESPX_BROKER_BIN` when starting compose.

## Start stack

```bash
docker compose -f deploy/broker/docker-compose.yaml up -d
```

Optional Sentinel overlay (coordination failover tests):

```bash
docker compose -f deploy/broker/docker-compose.yaml \
  -f deploy/broker/docker-compose.sentinel.yaml up -d
```

## Ports

| Listener | Address | Role |
| :--- | :--- | :--- |
| HAProxy produce | `0.0.0.0:9092` | Leader-only produce (`/leaderz` health check) |
| HAProxy any | `0.0.0.0:9093` | Any healthy broker (fetch / replication) |
| broker-1 TCP | `127.0.0.1:19093` | Direct gnet listener |
| broker-2 TCP | `127.0.0.1:19094` | Direct gnet listener |
| broker-1 health | `127.0.0.1:8081` | `/health`, `/healthz`, `/leaderz`, `/metrics` |
| broker-2 health | `127.0.0.1:8082` | same |
| HAProxy stats | `0.0.0.0:1936` | Backend up/down UI |
| Redis | `127.0.0.1:6379` | Coordination (lab-local instance) |

Broker nodes bind `19093`/`19094` so HAProxy can own `9093` without a port clash. Do not run this lab on the same host as compose `region-proxy` (`127.0.0.1:9093`).

## Health checks

- `/health` and `/healthz` — disk gate OK (503 when data dir is not writable)
- `/leaderz?topic=<name>` — 200 when this node is the topic leader

HAProxy leader backend uses `/leaderz?topic=tracker-logs`; the any-healthy backend uses `/healthz`.

## Fault lab

```bash
make test-broker-fault-lab
# or: bash scripts/fault/broker_fault_lab.sh
```

Broker throughput bench:

```bash
bash scripts/load/broker.sh
```

## Environment

See `.env.example` (`BROKER_*`, `ESPX_BROKER_BIN`, Sentinel vars). Lab compose uses a dedicated Redis on port `6379`; set `BROKER_REDIS_URL=redis://127.0.0.1:6379/0` when pointing tracker/processor at this stack.
