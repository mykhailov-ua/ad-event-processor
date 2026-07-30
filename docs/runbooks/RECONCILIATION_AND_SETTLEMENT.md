# Runbook: reconciliation, settlement, and drift detection

How eSPX ensures financial truth in a distributed system with asynchronous settlement.

**GAP:** GAP-HYG-30 — full acceptance criteria: [.cursor/GAP_SPECS.md](../../.cursor/GAP_SPECS.md#gap-hyg-30--pg-volume-meter--drift-audit--pinned-settlement).

Related: [ARCHITECTURE.md](../ARCHITECTURE.md), [DATA_SECURITY.md](./DATA_SECURITY.md), [PROTECTION.md](../PROTECTION.md).

---

## Problem: drift

Budget is checked in **Redis** (hot path) but recorded in **PostgreSQL** (cold path). ClickHouse is **analytics only**.

| Drift type | Cause |
| :--- | :--- |
| Redis ↔ PG | In-flight stream events; Redis ahead of ledger |
| PG ↔ CH | Settlement lag; lossy CH ingest; optional CH profile |
| Ledger | Cached `customers.balance` ≠ `SUM(balance_ledger)` |

**Goal:** zero financial loss; `AssertBudgetInvariant` ±1 micro-unit.

---

## Architecture

```text
Redis stream ──► SettlementWorker (pinned lanes) ──► PG txn ──► XACK
                        │                              │
                        │                              ├── balance_ledger
                        │                              ├── campaign_stats
                        │                              └── billing.usage_meters (GAP-HYG-30)
                        │
ReconWorker (5 min) ◄───┴── Redis spend keys vs PG vs CH
LedgerInvariantWorker (24 h) ── customers.balance vs ledger sum
```

### Pinned settlement lanes

| Rule | Detail |
| :--- | :--- |
| Lane | `crc32(campaign_id) % N` → dedicated goroutine |
| Ordering | One campaign never parallel-settled |
| Deadlock | `FOR UPDATE` on `customers` ordered by `customer_id` |
| Pools | `PgPoolSettle` (small, high priority) vs `PgPoolRead` (API) |

### Settlement batch

| Step | Action |
| :---: | :--- |
| 1 | `XREADGROUP` batch (100 ms or 1000 rows) |
| 2 | PG txn: idempotency check → ledger debit → stats bump → meter increment |
| 3 | Commit |
| 4 | `XACK` only after commit |

---

## ReconWorker audits

### Audit A — Redis ↔ PG (active spend)

| Field | Value |
| :--- | :--- |
| Check | `redis_spend + in_flight ≈ pg_ledger_spend` |
| Threshold | Configurable; default 0.1% or 1000 micro-units |
| Action | Force refill from PG; metric `ad_recon_drift_micro` |
| Alert | Notifier when refill runs |

### Audit B — PG ↔ CH (analytics parity)

| Field | Value |
| :--- | :--- |
| Check | `campaign_stats.spend` vs `CH sum(price)` |
| Tolerance | 0.01% |
| Action | `stale=true` on stats API; operator alert |
| Note | CH never billing authority |

### Audit C — ledger invariant

| Field | Value |
| :--- | :--- |
| Check | `customers.balance = SUM(balance_ledger.amount_micro)` |
| Cadence | 24 h full scan; sample hourly in ReconWorker |
| Action | `FORCE_PAUSE` all campaigns for customer via outbox |

---

## SLA (GAP-HYG-30)

| Metric | Target |
| :--- | :--- |
| Settlement batch commit p99 | < 500 ms |
| ReconWorker tick | < 500 ms |
| Global settle end-to-end p99 | < 2 s |
| Budget invariant | ±1 micro-unit after refill |
| Hot path | Untouched; 0 allocs |

---

## SQL plans

Detail and EXPLAIN targets: [GAP_SPECS § SQL — GAP-HYG-30](../../.cursor/GAP_SPECS.md#sql--gap-hyg-30).

| Query | Purpose | Index |
| :--- | :--- | :--- |
| `UpsertUsageMeter` | PG volume meter (replaces CH worker) | `events(created_at, status)` |
| Ledger invariant | Detect balance drift | `balance_ledger(customer_id)` |
| `sync_idempotency` insert | Prevent double debit | `UNIQUE(event_id, campaign_id)` |
| Campaign stats drift | PG vs CH compare input | `campaign_stats(campaign_id, date)` |

---

## Patterns

| Pattern | Use |
| :--- | :--- |
| Pinned consumer | Per-campaign ordering |
| Transactional batch | Ledger + stats + meter atomic |
| Idempotency key | `sync_idempotency` |
| Pool tiering | Settle vs read isolation |
| Outbox | `FORCE_PAUSE` side effects |
| Force refill | PG truth over Redis on crash |

---

## Tests

| Layer | Requirement |
| :--- | :--- |
| Unit | Lane hash distribution; drift threshold math |
| Integration | testcontainers PG+Redis; batch commit + XACK order |
| Fault | `TestFault_SettlementReplay`; `TestFault_ReconDriftRefill` |
| Proof | `fault_proof fault=recon_drift_within_band` |
| SQL | `EXPLAIN (ANALYZE, BUFFERS)` on meter rollup in `TestExplainAudit_*` |
| Invariant | `AssertBudgetInvariant` after force refill |

---

## Failure modes

| Failure | Mitigation |
| :--- | :--- |
| Redis crash | Replay from last `XACK`; idempotency prevents double debit |
| PG txn timeout | Retry batch; no `XACK` |
| Double spend | `sync_idempotency` unique constraint |
| Negative balance | Lua blocks hot path; settlement logs + alert |
| CH down | Metering and billing continue on PG; stats `stale=true` |
