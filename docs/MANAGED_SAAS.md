# Managed SaaS cells (vendor-hosted)

Optional product line separate from self-hosted on-prem SKUs. Phase 1 is **one isolated compose project per buyer**, not a shared multi-tenant control plane.

## What ships in phase 1

| Piece | Location |
| :--- | :--- |
| JWT claim `deployment_mode` (`on_prem` default, `managed_saas` for vendor cells) | `internal/licensing/deployment_mode.go` |
| SKU `managed_saas` | `deploy/vendor/sku.yaml` |
| Vendor bootstrap | `scripts/install/managed_saas_cell_bootstrap.sh` |
| Compose overlay | `deploy/compose/docker-compose.managed-saas-cell.yaml` |
| License status field | `GET /api/v1/license/status` → `deployment_mode` |

Self-hosted clones (`README.md`, on-prem SKUs) are unchanged. Issue `managed_saas` JWT only for vendor-operated stacks.

## Provision a cell

```bash
bash scripts/install/managed_saas_cell_bootstrap.sh \
  --cell-id acme-west \
  --customer "Acme Media"
```

This writes:

- `var/saas-cells/<cell_id>/license.jwt` (SKU `managed_saas`)
- `var/saas-cells/<cell_id>/.env` with `COMPOSE_PROJECT_NAME=saas-<cell_id>`

Start (or let bootstrap start with default):

```bash
export COMPOSE_PROJECT_NAME=saas-acme-west
export DEPLOYMENT_MODE=managed_saas
docker compose \
  -f deploy/compose/docker-compose.yaml \
  -f deploy/compose/docker-compose.managed-saas-cell.yaml \
  --profile single_vps up -d
```

Each cell has dedicated Docker volumes (project name isolation). Hot path still uses in-memory registry snapshots; no per-request PG on `/track`.

## Data residency and export

Financial truth: Postgres `balance_ledger`. Operational usage: `billing.usage_daily` (not invoice).

| Export | Route | Notes |
| :--- | :--- | :--- |
| Usage meters CSV | `GET /api/v1/billing/usage/export?format=csv&from=&to=` | Per-workspace; optional `cost_center` filter |
| Ledger async job | `POST /api/v1/billing/exports` | Chunk size from JWT `max_export_chunk_bytes` |
| Ledger CSV page | `GET /api/v1/customers/{id}/balance/export?format=csv` | Cursor via `X-Next-Cursor` |

Offboarding runbook (vendor):

1. Run usage + ledger exports for the buyer `customer_id` range.
2. `pg_dump` the cell Postgres (`db` service) and ClickHouse backup per `docs/DEVELOPMENT.md`.
3. `docker compose down -v` on the cell project after handoff window.

Region pinning is deploy-time (compose host/region choice), not a separate JWT field.

## Not in phase 1

- Shared control plane with schema-per-tenant routing
- Per-tenant hot-path registry partition in one process
- k8s namespace orchestration (compose-per-customer only)

Cross-tenant IDOR holdouts: `internal/controlplane/api_fault_test.go`, `workspace_billing_test.go`.
