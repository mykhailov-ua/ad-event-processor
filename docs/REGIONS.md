# Multi-region deployment (Enterprise)

**Not appliance SKU.** Requires Enterprise license (`features.multi_region`), installer profile, compose profile `multi-region`. See [ARCHITECTURE.md](ARCHITECTURE.md) §11.

## Components

| Piece | Path | Role |
| :--- | :--- | :--- |
| `region-proxy` | `cmd/region-proxy`, `pkg/regionproxy/` | Regional WAL ingress, quorum book/ack, uplink |
| Regional processor | `cmd/processor` + `MULTI_REGION_ENABLED=1` | Spend sync → region-proxy |
| Global control | `control` | `IngestRegionProxyBatch`, operation leases, global PG |
| Compose | `deploy/compose/docker-compose.yaml` | `--profile multi-region` |

Pilot: `multi_region: false` in `deploy/vendor/sku.yaml`.

## Environment (regional cell)

| Variable | Purpose |
| :--- | :--- |
| `MULTI_REGION_ENABLED=1` | Regional processor mode |
| `REGION_CODE` | Cell identifier |
| `REGION_PROXY_ADDR` | region-proxy listen |
| `REGION_PROXY_REDIS_URL` | Optional proxy metadata Redis |
| `GLOBAL_SPEND_BATCH_MIN` | Min batch before flush (default 100) |

```bash
docker compose -f deploy/compose/docker-compose.yaml --profile multi-region up -d
bash scripts/dev/stack.sh --profile multi-region
```

## Region-proxy WAL

Append-only WAL (`pkg/regionproxy/wal/`): group commit (`fsyncSem` capacity 1), crash recovery via `Recover()` (discards torn records). Quorum: `pkg/regionproxy/quorum` (operation leases when multi-region enabled).

## 90-minute resilience drill

```bash
bash scripts/test/mr_resilience_drill.sh
```

1. **0–10 min:** p99 &lt; 80 ms; node weights healthy.
2. **10–20 min:** MR fault suite — ≥12 `mr_*` `fault_proof` lines.
3. **20–35 min:** Quorum partition — only 1/3 region-proxy active; global updates blocked.
4. **35–50 min:** Stop regional PG; lease expires; local budget continues.
5. **50–65 min:** Kill region-proxy replica; RTO &lt; 120 s.
6. **65–75 min:** Pause global PG 60 s; trackers online; proxies spool WAL.
7. **75–85 min:** `AssertBudgetInvariant` — Redis vs PG ±1 micro-unit.
8. **85–90 min:** Teardown, outboxes drained.

Unit proofs: `go test -run 'TestFault_(Score|OperationLease|Region|Proxy|Disk|Global|Quorum)' ./internal/controlplane/...`

E2E: `tests/e2e/region_proxy_uplink_test.go`, `region_proxy_ingress_test.go`.

## Import boundaries

Hot-path `internal/ingestion` (non-`_test`) does **not** import `pkg/regionproxy`. `cmd/processor` wires region-proxy client only when `MULTI_REGION_ENABLED=1`. Cold-path `operation_lease.go` uses `pkg/regionproxy/quorum` when Enterprise multi-region enabled.
