# Service Boundaries

Service boundaries for control-plane binaries under `cmd/*`. The ingestion hot path (`tracker`, `processor`, stream subscribers) must not absorb unrelated domains.

---

## Part 1 - Current Platform Layout

### Database

Control-plane services share one Postgres instance in compose (default DB name `ad_event_processor` in `.env`; rename to `espx` is cosmetic).

1. Per-domain schemas: `payment`, `billing`, `notifier`.
2. Shared public schema: `campaigns`, `events`, `balance_ledger`, `outbox_events`, `sync_idempotency`.
3. Auth tables: `users`, `sessions`, `api_keys` in public schema.

Isolation is at schema and idempotency-key-prefix level, not separate Postgres containers by default.

### Control-plane binaries

| Binary | Role | Ports (typical) |
| :--- | :--- | :--- |
| `cmd/management` | Admin HTTP, settlement gRPC, outbox/recon/pacing workers | :8188, :51053 |
| `cmd/auth` | Auth gRPC | :51051 |
| `cmd/payment` | Payment gRPC, Stripe webhooks, HTMX checkout | :51052, :8187 |
| `cmd/billing` | Invoice gRPC | :51054 |
| `cmd/notifier` | Notification gRPC + delivery worker | :8085 |
| `cmd/ivt-detector` | ClickHouse batch analyzer -> management blacklist API | batch job |
| `cmd/log-evacuator` | Local tracker log archive (compose `tools` profile) | none |

Settlement gRPC runs inside `management` (no standalone `cmd/settlement`).

### Scaling levels when adding services

1. New domain: `CREATE SCHEMA <svc>`, migrations under `internal/<svc>/migrations/`, shared `DB_DSN`, gRPC + optional `x-internal-token`.
2. Connection pool limits per service in compose and config.
3. Read-heavy analytics: indexes on shared Postgres; ClickHouse for IVT/recon.
4. Compliance blast radius: separate Postgres container (e.g. payments only) as last resort.

### Per-service notes

### `internal/ivtdetector` - ClickHouse analytics -> idempotency claims -> blacklist via management HTTP. Separate binary from admin process.

### `internal/notifier` - gRPC queue -> `notifier.notifications` -> background delivery. Standalone `:8085`.

### `internal/logevacuator` - Rotated tracker segments `*.log.zst.ready` -> local archive with checkpoint file. Node-local I/O; no RPC. Same host as tracker logs only.

### `internal/payment` - Isolated webhook port, Stripe credentials, outbox -> management settlement -> `balance_ledger`. Keep separate: webhook storms and provider outages isolated from admin.

### `internal/billing` - Month-end invoice generation from `balance_ledger`; schema `billing`. Keep separate: read-heavy batch profile.

---

## Part 2 - Decision Framework

Use when adding a domain or splitting/merging `cmd/*` binaries.

### Workload classification

| Class | Examples | Policy |
| :--- | :--- | :--- |
| Hot-path | Tracker parse/filter, stream ingest | Never split; never mix domains |
| Control-plane | Auth, payment, billing, admin REST | Standalone if score >= 11 |
| Batch/cron | IVT scans, partition janitors, recon | Library + worker in existing cold binary |
| Node utility | Log evacuator, local backups | Standalone near data; no network API |

Batch/cron and node utility: do not create a microservice without shared RPC clients.

### Criteria matrix (0-2 each, max 18)

| # | Criterion |
| :--- | :--- |
| H | Hot-path isolation risk if co-located |
| E | External network (webhooks, OAuth, public gateways) |
| S | Secret/compliance isolation (PCI, gateway keys) |
| F | Failure blast radius (retry storms, OOM on admin) |
| L | Load profile mismatch (polling, bursty webhooks) |
| C | Caller count (multiple independent gRPC clients) |
| D | Data ownership (dedicated schema + migrations) |
| O | Operational independence (optional in compose) |
| T | Team/lifecycle separation |

| Score | Outcome |
| :--- | :--- |
| 0-5 | Monolith package in `internal/<domain>/` inside management or processor |
| 6-10 | Modular monolith: package + background goroutine |
| 11+ | Standalone `cmd/<service>` |

Apply veto rules regardless of score.

### Veto rules

- Never inject into `tracker`: reflection pools, cron loops, Postgres writes not required for ingest.
- Never call external microservices from `processTrack()` or filter pipeline.
- Do not split without active callers or webhooks; use library + worker until integration is live.
- Do not split for layout only when DB is already shared across schemas.
- Node-local I/O (logs, mmap broker, checkpoints): local utility binary, not network service.

### Decision tree

1. Tracker hot path -> `internal/ingestion` library only.
2. Node-local files -> standalone utility (`tools` profile).
3. External webhooks or credentials -> standalone service.
4. Multiple callers or optional deploy -> score criteria; >= 11 -> standalone service; else package + worker in management/processor.

### New standalone checklist

1. `internal/<svc>/` + `cmd/<svc>/main.go`
2. `api/<svc>.proto` via buf/vtproto
3. Migrations with `CREATE SCHEMA <svc>`; shared `DB_DSN`
4. Config in `internal/config/`; isolated pool limits
5. `x-internal-token` on service-to-service calls
6. Compose entry with memory limits; ports in `../ARCHITECTURE.md`
7. At least one active client before enforcing in compose

### Monolith package checklist

1. Package under `internal/<domain>/`
2. Worker in `cmd/management/main.go` or `cmd/processor/main.go`
3. Reuse host pool or strict private pool limit
4. No gRPC/HTTP unless external systems require it

### Classification summary (reference scores)

| Service | Score | Notes |
| :--- | ---: | :--- |
| management | 10/18 | Hub; settlement gRPC co-located |
| auth | 14/18 | Standalone |
| payment | 16/18 | Standalone |
| billing | 11/18 | Standalone |
| notifier | 12/18 | Standalone |
| ivt-detector | 7/18 | Standalone batch |
| log-evacuator | 8/18 | Standalone utility |

Open product work: [OPEN_GAPS.md](./OPEN_GAPS.md).

### Related

- [../ARCHITECTURE.md](../ARCHITECTURE.md) - topology, settlement
- [../DEVELOPMENT.md](../DEVELOPMENT.md) - open gaps
- `.cursor/rules/espx.mdc` - hot-path constraints
