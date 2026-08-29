# deploy

Runtime and infrastructure artifacts. This tree is **not** application logic — compose files, edge configs, Lua, Redis scripts, ClickHouse DDL, BPF sources, bundled schemas, and vendor license material.

Cross-ref: [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md), [scripts/SCRIPTS.md](../scripts/SCRIPTS.md), `.cursor/rules/edge.mdc`, `.cursor/rules/data-layer.mdc`.

---

## Role in the system

| Layer | deploy/ owns | Processes that consume it |
| :--- | :--- | :--- |
| Perimeter | `nginx/`, `edge/` | OpenResty `:8180`, optional `edge-xdp`, `edge-bpf-sync` |
| Data plane config | `redis/`, `clickhouse/` | tracker, processor, control |
| Orchestration | `compose/` | `scripts/dev/stack/stack.sh`, CI fault drills |
| Integrations | `schemas/`, `feeds/` | control, traffictemplates codegen |
| Install | `installer/`, `systemd/` | `scripts/install/*` |
| Observability | `monitoring/` | Prometheus/Grafana overlays |
| Commercial | `vendor/` | license-issue, operator docs |

---

## Directory map

### `compose/`

Docker Compose stacks and overlays.

| File | Use |
| :--- | :--- |
| `docker-compose.*.yaml` | Base profiles: control-dev, memory-dev, tracker-local, load-test |
| Overlays | Memory cgroups (`memory-dev`), CPU isolation, analytics-ml |

**Operate:**

```bash
cp .env.example .env
bash scripts/dev/stack/stack.sh ingest-only   # laptop default — no ClickHouse
bash scripts/dev/stack/stack.sh full          # tracker + processor + control + PG + Redis
CH_ENABLED=1 bash scripts/dev/stack/stack.sh clickhouse   # explicit CH opt-in
bash scripts/dev/stack/stack.sh down
```

**Limits:**

- Local default `REDIS_SHARD_COUNT=2`; production expects 4 shard addresses in `REDIS_ADDRS`.
- `COMPOSE_MEMORY_PROFILE=dev` applies reduced cgroup limits — required on 16 GB laptops.
- Do **not** leave ClickHouse running overnight on dev machines (~4 GB compose budget).
- Root `docker-compose.yaml` + `docker-compose.override.yaml` are the entry; stack.sh appends overlays.

**Test:** `bash scripts/dev/stack/preflight.sh` after up. Fault tier: `bash scripts/fault/compose_fault_drill.sh all`.

---

### `nginx/`

OpenResty edge: rate limits, circuit breaker, blacklist shared dict, body DFA, shard routing to tracker pool.

| Path | Role |
| :--- | :--- |
| `lua/access-check.lua` | Request gate before upstream |
| `lua/edge_track_policy.lua` | Track-specific policy |
| `lua/edge-blacklist-sync.lua` | Redis shard 0 → `ngx.shared.blacklist_cache` |
| `lua/edge-slot-map.lua` | Must match Go `StaticSlotSharder` slot table |

**Operate:** Edge listens `:8180` / `:443`. Tracker upstreams `8181-8184` by campaign slot.

**Limits:**

- Lua slot map and Go sharding must stay in sync — parity tests in `internal/domain/`.
- `/track` wire policy: `Content-Length` required, chunked TE rejected (nginx ↔ gnet chaos tests).
- CDN in front without TCP/TLS fingerprint sync: disable or expect fail-open on MSS/TTL/JA3 signals (`deploy/vendor/ANTIFRAUD.md`).

**Test:**

```bash
bash scripts/test/edge/lua_tests.sh all
go test ./internal/ingest/ -run TestChaos_CrossHop_NginxGnet -count=1
```

---

### `redis/`

Unified-filter Lua, budget scripts, topology references.

| Artifact | Role |
| :--- | :--- |
| `unified-filter.lua` | Atomic debit, dedup, pacing — **one `EVALSHA` per accepted event** |
| `budget-rollback.lua` | Post-debit enqueue failure rollback |
| `local-quota-refill.lua` | Local quanta refill |
| Sentinel configs | HA — not Redis Cluster |

**Critical invariants:**

- All campaign keys use `{campaign_id}` hash tag on **one master** per shard.
- Never add a second synchronous Redis round-trip on the hot path for the same event.
- Stream `XADD` is **not** inside unified-filter Lua — async `StreamProducer` / `BrokerProducer` only.
- Shard 0 holds pub/sub `campaigns:update`, global blacklists, BPF sync source.

**Test:** `make test-integration` (testcontainers Redis). `bash scripts/ops/verify_redis_topology.sh .env`.

---

### `clickhouse/`

| File | Role |
| :--- | :--- |
| `init.sql` | Bootstrap DDL — must match `internal/clickhouse/migrate/00000_bootstrap_tables.sql` |
| `config.load-test.yaml` | Memory-capped CH for load-test compose |

**Limits:** CH is cold path only. Processor ingests streams → CH. Reports and IVT/ML read CH — not tracker.

**Test:** CH integration tests in `tests/integration/`. Enable locally only when reproducing analytics bugs.

---

### `edge/`

eBPF/XDP C sources and attach specs.

| Component | Binary | License |
| :--- | :--- | :--- |
| XDP program | `cmd/edge-xdp` | `ebpf_xdp_edge` (Enterprise) |
| Map sync | `cmd/edge-bpf-sync` | same |

**Operate:** Build BPF artifacts: `make gen bpf-dev`. Sync Redis blocklists into LRU/LPM maps.

**Limits:** XDP drops known L3/L4 bad IPs and syn-flood patterns — not residential proxy rotation or app-layer fraud. LRU eviction under map pressure is possible — monitor `edge_blocklist_map_fill_ratio`.

**Test:** `go test ./internal/edge/...`. Manual: `bash scripts/test/edge/xdp_resilience_drill.sh` (enterprise workflow).

---

### `schemas/`

Bundled integration YAML (traffic click schemas, affiliate templates, onboarding catalog). Consumed by `make gen` / `cmd/codegen-traffic-templates` and admin import APIs.

**Test:** `go test ./internal/traffictemplates/...`, `go test ./internal/integrationschema/...`.

---

### `broker/`

Mmap WAL layout references for `pkg/broker` and `cmd/broker`.

**Operate:** `CH_INGEST_SOURCE=broker` — shadow mode first (`BROKER_SHADOW_MODE=1`), then cutover. Redis streams remain for budget Lua.

**Test:** `go test ./pkg/broker/...`, `go test ./internal/ingest/ -run TestFault_Broker -count=1`.

---

### `monitoring/`

Prometheus scrape templates, Grafana panel sources. Wire into load-test compose via `make load-test-config`.

---

### `geoip/`

GeoIP fetch placeholders. Operators download MaxMind (or equivalent) per [docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md).

---

### `installer/`, `systemd/`, `operator/`

Appliance install assets and unit templates. Entry: `bash scripts/install/appliance_bootstrap.sh`.

---

### `vendor/`

License keys, SKU, sales/marketing/antifraud docs, pentest fixtures. **Do not ship in customer packages.**

---

## Topology quick reference

```
Client → nginx (:8180) → tracker (:8181-8184) → Redis
                        → control (:8188) → Postgres
Redis streams → processor (:8186) → Postgres, ClickHouse
```

| Service | Default port | Metrics |
| :--- | ---: | ---: |
| nginx edge | 8180, 443 | edge exporter |
| tracker | 8181-8184 | 9101-9104 |
| processor | 8186 | 9106 |
| control | 8188 | 9108 |
| payment webhooks | 8187 | — |
| Postgres | 5430 | — |
| Redis masters | 6479-6482 | — |
| ClickHouse | 9000 | — |

---

## Production constraints

| Constraint | Detail |
| :--- | :--- |
| Redis topology | Standalone masters + Sentinel — **not** Redis Cluster on hot path |
| Shard parity | Edge Lua slot map = Go `CRC32C(campaign_id) & 1023` |
| CH optional for ingest | `CH_ENABLED=0` — processor skips CH sink; IVT/ML/reports need CH |
| License file | `var/license.jwt` — Ed25519 JWT, offline verify |
| Secrets | Never commit keys; `deploy/vendor/license_private.key` is gitignored |
| Sysctl | High connection counts — `deploy/sysctl/`, `scripts/ops/sysctl.sh` |

---

## Development pitfalls

1. **Editing `init.sql` without migration** — sync `internal/clickhouse/migrate/` in the same commit.
2. **Changing Lua without integration tests** — budget invariant and dual-XADD regressions are subtle.
3. **Starting full stack on laptop** — use `ingest-only`; CH only when needed.
4. **Mismatching slot tables** — silent wrong-shard debits; always run sharding tests after edge changes.
5. **Hand-editing bpf2go output** — regenerate via `make gen bpf-dev`.
6. **Compose profile drift** — document new services in `stack.sh` help and this file.

---

## Verification checklist

| Check | Command |
| :--- | :--- |
| Stack health | `bash scripts/dev/stack/preflight.sh` |
| Redis topology | `bash scripts/ops/verify_redis_topology.sh .env` |
| Edge Lua | `bash scripts/test/edge/lua_tests.sh all` |
| Parser parity | `go test ./internal/ingest/ -run TestChaos_CrossHop -count=1` |
| Fault compose | `bash scripts/fault/compose_fault_drill.sh all` |
| CH DDL parity | diff `deploy/clickhouse/init.sql` vs bootstrap migration |
