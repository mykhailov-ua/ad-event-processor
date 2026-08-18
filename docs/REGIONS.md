# Multi-region deployment (Enterprise)

**Not part of appliance SKU.** Enable only with Enterprise license (`features.multi_region`), installer profile, and compose profile `multi-region`.

See also: [ARCHITECTURE.md](ARCHITECTURE.md) section 11.

---

## Components

| Piece | Path | Role |
| :--- | :--- | :--- |
| `region-proxy` | `cmd/region-proxy`, `pkg/regionproxy/` | Regional WAL ingress, quorum book/ack, uplink to global control |
| Regional processor | `cmd/processor` with `MULTI_REGION_ENABLED=1` | Spend sync batches → region-proxy (`REGION_PROXY_ADDR`) |
| Global control | `control` | `IngestRegionProxyBatch`, operation leases, global PG truth |
| Compose | `deploy/compose/docker-compose.yaml` | Services under `--profile multi-region` only |

License: pilot `multi_region: false` in `deploy/vendor/sku.yaml`. Runtime gate in `internal/controlplane/serve.go`.

---

## Environment (regional cell)

| Variable | Purpose |
| :--- | :--- |
| `MULTI_REGION_ENABLED=1` | Regional processor mode |
| `REGION_CODE` | Cell identifier |
| `REGION_PROXY_ADDR` | `region-proxy` listen address |
| `REGION_PROXY_REDIS_URL` | Optional Redis for proxy metadata |
| `GLOBAL_SPEND_BATCH_MIN` | Min batch before spend-sync flush (default 100) |

Stack with profile:

```bash
docker compose -f deploy/compose/docker-compose.yaml --profile multi-region up -d
# or dev stack:
bash scripts/dev/stack.sh --profile multi-region
```

---

## Region-proxy WAL

Regional processors write events to an append-only WAL (`pkg/regionproxy/wal/`):

- **Group commit**: `fsyncSem` capacity 1 serializes `fsync`; concurrent appends share one sync.
- **Crash recovery**: `Recover()` scans tail, discards torn records, remaps segment before accepting traffic.

Quorum helpers live in `pkg/regionproxy/quorum` (used by `control` operation leases when multi-region is enabled).

---

## 90-minute operator resilience drill

Runner:

```bash
bash scripts/test/mr_resilience_drill.sh
```

Optional CI: `.github/workflows/enterprise-resilience.yaml` (`workflow_dispatch`).

### Checklist

1. **Min 0–10 (Baseline)**: p99 &lt; 80 ms; node weights healthy.
2. **Min 10–20 (Fault injection)**: MR fault suite — at least 12 `mr_*` `fault_proof` lines (subset of `run_resilience.sh`).
3. **Min 20–35 (Quorum)**: Partition so only 1/3 region-proxy replicas active; global updates blocked.
4. **Min 35–50 (Lease partition)**: Stop regional PostgreSQL; lease expires; local budget spend continues.
5. **Min 50–65 (Proxy failover)**: Kill a `region-proxy` replica; RTO **&lt; 120 s**.
6. **Min 65–75 (Global DB outage)**: Pause global PostgreSQL 60 s; trackers online; proxies spool WAL.
7. **Min 75–85 (Invariants)**: `AssertBudgetInvariant` — Redis vs PG within ±1 micro-unit.
8. **Min 85–90 (Teardown)**: Logs, restore config, outboxes drained.

Unit MR proofs (no full geo stack): `go test -run 'TestFault_(Score|OperationLease|Region|Proxy|Disk|Global|Quorum)' ./internal/controlplane/...`

E2E (often skipped in CI): `tests/e2e/region_proxy_uplink_test.go`, `region_proxy_ingress_test.go`.

---

## Import boundaries (appliance)

Production hot-path ingestion (`internal/ingestion`, non-`_test`) does **not** import `pkg/regionproxy`. Spend sync uses `SpendSyncTransport` interface; `cmd/processor` wires `region-proxy` client only when `MULTI_REGION_ENABLED=1`.

Cold-path `internal/controlplane/operation_lease.go` still uses `pkg/regionproxy/quorum` when Enterprise multi-region is enabled — acceptable until lease store is abstracted.
