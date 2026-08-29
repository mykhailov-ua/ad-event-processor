# cmd

Process entrypoints. **Keep `main.go` thin** — parse flags/env, call `wire.go` or `internal/control/deps.go`, start servers. No business logic in `main`.

Cross-ref: [internal/INTERNAL.md](../internal/INTERNAL.md), [deploy/DEPLOY.md](../deploy/DEPLOY.md).

---

## Hot path binaries

### `tracker`

| | |
| :--- | :--- |
| **Ports** | 8181-8184 (one per pinned worker / shard lane) |
| **Metrics** | 9101-9104 |
| **Routes** | `/track`, `/click`, `/openrtb/bid`, `/tg/*`, `/static/track.js` |
| **Wiring** | `cmd/tracker/wire.go` |

**Responsibilities:**

- gnet HTTP server — **not** `net/http` on ingest path.
- `FilterEngine` chain ending in `UnifiedFilter` (Redis Lua).
- `StreamProducer` / `BrokerProducer` async enqueue.
- Campaign registry from `atomic.Pointer` snapshot + Redis pub/sub reload.

**Operate:**

```bash
go build -o bin/tracker ./cmd/tracker/
# or via compose:
bash scripts/dev/stack/stack.sh ingest-only
```

**Hard limits (do not violate in code wired here):**

| Rule | Why |
| :--- | :--- |
| No Postgres/ClickHouse on request thread | p99 SLA |
| No ML inference | `internal/fraud` is cold only; read boost snapshot |
| At most **one** sync Redis `EVALSHA` per accept | Multi-RTT kills budget |
| `TryReserve` before Lua debit | Admission race — spend without sink |
| Filter `Check` in detached goroutine | gnet workers must not block on Redis |
| Zero heap allocs on `/track` | `make test-alloc-gate` |
| No `fmt.Sprintf`, `interface{}`, `context.With*` on hot path | CI static gates |

**Test:**

```bash
make test-alloc-gate
go test ./internal/ingest/ -short -count=1
go test ./internal/ingest/ -run TestFault_ -count=1   # fault tier
bash scripts/test/load/gate_run.sh                     # perf tier
```

---

### `processor`

| | |
| :--- | :--- |
| **Port** | 8186 |
| **Metrics** | 9106 |
| **Role** | Consume Redis streams / broker → Postgres, ClickHouse |

**Responsibilities:**

- Batch settlement, conversion smart reject, fraud event routing.
- Does **not** write `balance_ledger` (payment/billing handlers do).

**Operate:** Started by compose with tracker. Requires Redis streams or broker consumer configured.

**Test:** `go test ./internal/stream/...`, `make test-integration`, `make test-fault`.

---

## Control plane

### `control`

| | |
| :--- | :--- |
| **Port** | 8188 |
| **Metrics** | 9108 |
| **Routes** | `/api/v1/*`, admin static stub, outbox workers in-process |

**Responsibilities:**

- Modular monolith: admin API, payment webhooks (`:8187` may be separate listener), outbox poll (~20 ms), billing, reports wiring.
- Bootstrap: `internal/control/deps.go`.

**Operate:**

```bash
go build -o bin/control ./cmd/control/
bash scripts/dev/stack/seed_admin.sh   # bootstrap operator
```

**Limits:**

- Admin mutation + `outbox_events` in **same PG transaction**.
- No direct Redis writes from handlers — outbox worker applies side effects.
- Cold-path body limit 64 KiB (`pkg/coldpath`).

**Test:**

```bash
go test ./internal/controlplane/ -short -count=1
bash scripts/ci/admin/openapi.sh
go test ./internal/controlplane/ -run TestAdminStaticRoutes -count=1
```

---

## Edge and broker

| Binary | Role | When to run |
| :--- | :--- | :--- |
| `edge-xdp` | Attach XDP, NIC-level drops | Enterprise license, dedicated edge host |
| `edge-bpf-sync` | Redis → BPF maps | With `edge-xdp` |
| `broker` | mmap WAL ingest broker | `CH_INGEST_SOURCE=broker` |
| `region-proxy` | Multi-region WAL/quorum | `multi_region` license |

**Test:** `go test ./pkg/broker/...`, fault tests for shadow cutover.

---

## Analytics / ML sidecars

| Binary | Role | Profile |
| :--- | :--- | :--- |
| `ivt-detector` | CH rules → outbox (blacklist, boost) | `analytics-ml` + Pro SKU |
| `fraud-scorer` | Batch LightGBM → Redis boost keys | `analytics-ml` + Scale SKU |
| `postback-sender` | Outbound postback delivery | tools profile |

**Limits:** Pauses when outbox `PENDING` > 500. Never wire ML into `tracker`.

---

## Operator / vendor tools

| Binary | Role |
| :--- | :--- |
| `admin` | DB seed, catalog, slot map CLI |
| `operator` | Operator maintenance |
| `license-issue` | Issue Ed25519 JWT |
| `license-asset-seal` | Seal assets with MCK |
| `trial-registry` | Trial registry CRUD |
| `vendor-trial-bot` | Telegram trial intake |
| `dlq` | DLQ inspect/replay |
| `migrate-cold-path` | Cold-path migration helper |
| `openapi-export` | Export OpenAPI bundle |
| `codegen-traffic-templates` | Regenerate traffic templates JSON |
| `patch-vtproto-hotpath` | Patch vtproto for 0-alloc parse |
| `installer` | Install helper binary |

---

## Load / perf / observability

| Binary | Role |
| :--- | :--- |
| `loadgen` | HTTP load generator |
| `load-report` | Parse BPF/load gate reports |
| `perf-gate` | Local perf gate runner |
| `bpf-collector` | BPF probe aggregation |
| `log-shipper`, `log-evacuator`, `log-compactor` | Log pipeline nodes |
| `alertmanager-telegram` | Alertmanager → Telegram bridge |
| `ml-replay`, `ml-validate` | ML offline tools |
| `campaign-shard` | Shard utility |

---

## Wiring conventions

```
cmd/<binary>/
  main.go      # flags, signal handling, call wire
  wire.go      # construct deps (tracker, broker, control)
  doc.go       # package main (may be empty)
```

| Rule | Detail |
| :--- | :--- |
| `main.go` ≤ 150 lines | Move wiring to `wire.go` |
| No SQL in `cmd/` | belongs in `internal/<domain>/` |
| Config from env | `internal/config` — single parse path |
| Metrics | Register in domain package, expose on dedicated port |

---

## Build matrix

```bash
go build -o bin/tracker ./cmd/tracker/
go build -o bin/processor ./cmd/processor/
go build -o bin/control ./cmd/control/
```

Release (garbled): `bash scripts/ci/release_garble.sh` — see [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md).

---

## Common mistakes

1. **Adding PG call in tracker `wire.go` init** — config snapshot only; cold reload via pub/sub.
2. **Running fraud-scorer without CH** — IVT/ML need `analytics-ml` profile.
3. **Multiple tracker instances without slot awareness** — each listener must match edge upstream map.
4. **Skipping license file** — tracker fails closed in production when `LICENSE_REQUIRED`.
