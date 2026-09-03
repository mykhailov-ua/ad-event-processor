# Enterprise deployment: XDP edge and multi-region

Operator reference for optional Enterprise surfaces. Not shipped in customer appliance packages. License matrix: [sku.yaml](./sku.yaml). Compose layout: [deploy/DEPLOY.md](../DEPLOY.md).

---

## License gates

| Surface | JWT feature | Minimum SKU | Compose profile |
| :--- | :--- | :--- | :--- |
| XDP edge | `ebpf_xdp_edge` | `enterprise` | `enterprise-xdp` |
| Multi-region | `multi_region` | `network` (up to 3 regions) or `enterprise` (up to 99) | `multi-region` |

Pilot and `single_vps` profiles exclude both. Entitlements are mirrored to Redis `entitlement:deployment` on license apply; edge daemons re-check every tick.

---

## Part 1: XDP edge

### Role

NIC-level L4 filter on the ingress interface. Drops known bad IPs, SYN/PPS floods, and non-TCP noise targeting the tracker port before packets reach nginx. Parallel L7 perimeter remains on OpenResty (`:8180`, `:443`).

XDP is **not** a substitute for parser budgets, unified-filter Lua, or app-layer fraud. Rotating residential proxies and CDN-terminated TCP evade host maps.

### Components

| Binary | Path | Role |
| :--- | :--- | :--- |
| `edge-xdp` | `cmd/edge-xdp` | Attach BPF programs; pin maps under `BPF_PIN_DIR` |
| `edge-bpf-sync` | `cmd/edge-bpf-sync` | Redis shard 0 -> pinned maps; ringbuf drain; metrics |
| `edge_filter.c` | `deploy/edge/xdp/bpf/` | `xdp_edge_filter`, `xdp_syn_cookie` programs |

Single container image: `deploy/edge/xdp/Dockerfile`. Entrypoint starts `edge-xdp`, waits for pinned `blocklist_v4`, then execs `edge-bpf-sync`.

### Features

| Feature | Behavior |
| :--- | :--- |
| IPv4/IPv6 blocklist | Host LRU HASH + LPM TRIE; allowlist checked before deny |
| SYN rate limit | PERCPU_HASH per source; global SYN cap; /24 subnet LRU aggregate |
| PPS / RST token bucket | PERCPU in-place updates; sub-second credit refill |
| SYN cookie | Optional (`XDP_SYN_COOKIE=1`); tail-call on limit breach; outbound SYN-ACK `doff=5` |
| Violations ringbuf | Drop reasons -> autoban on Redis (`AUTOBAN_TTL` default 5m) |
| TCP fingerprint | Optional (`XDP_FINGERPRINT=1`); SYN fields -> Redis `edge:tcp_fp:*` -> nginx headers |
| ICMP | PMTUD Type 3 Code 4 `XDP_PASS`; other ICMP on tracker path dropped |
| Non-TCP | UDP/SCTP to tracker port dropped; other UDP passed |

Map capacity (from `edge_filter.c`): blocklist host/LPM up to 786432 entries; allow LPM 65536; ringbufs 256 KiB each.

### Data flow

```
control outbox / fraud worker / operator block
  -> Redis shard 0 (blacklist:manual|auto|fraud, changelog ZSETs)
  -> edge-bpf-sync (full SMEMBERS every 5 min + incremental changelog)
  -> pinned BPF maps
  -> edge-xdp (lookup only on deny path)
  -> nginx -> tracker
```

Parallel L7 path: `edge-blacklist-sync.lua` -> `ngx.shared` generational `_bl_ver`. Rollback XDP without losing L7 blocks.

### Host requirements

| Requirement | Detail |
| :--- | :--- |
| Kernel | Linux 6.1+ with BTF (`/sys/kernel/btf/vmlinux`) |
| Capabilities | `CAP_BPF`, `CAP_NET_ADMIN` |
| Container | `privileged: true`, `network_mode: host`, mount `/sys/fs/bpf` and BTF read-only |
| NIC | Dedicated ingress interface; not behind L4 LB that hides client IPs |
| Build | `make gen bpf-dev` before image build if BPF objects not generated |

`EDGE_XDP_MODE`: `generic` (default, widest compat), `native`, `offload` (driver-dependent fallback chain in `edge-xdp`).

### Deploy (compose lab)

```bash
cp .env.example .env
# Set EDGE_XDP_INGRESS_INTERFACE to real NIC in production (default lo is lab only).
make gen bpf-dev
docker compose -f deploy/compose/docker-compose.yaml --profile enterprise-xdp up -d edge-xdp
```

Apply Enterprise JWT with `ebpf_xdp_edge: true` before expecting attach and sync loops.

### Deploy (bare metal / installer)

1. Issue Enterprise license; apply via Admin or `POST /api/v1/license/apply`.
2. Build BPF: `make gen bpf-dev`.
3. Install binaries: `edge-xdp`, `edge-bpf-sync` (from release image or `go build`).
4. Set `INGRESS_INTERFACE` (see `deploy/edge/nic-tune.env.example` for IRQ tuning).
5. Optional: `bash scripts/install/ad-event-processor-install.sh apply` with `edge_xdp: true` in install manifest.
6. Start `edge-xdp` then `edge-bpf-sync` (or use container entrypoint).

Pin directory default: `/sys/fs/bpf/ad-event-processor`.

### Environment

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `INGRESS_INTERFACE` | required | NIC name for XDP attach |
| `EDGE_XDP_INGRESS_INTERFACE` | `lo` | Compose alias for above |
| `BPF_PIN_DIR` / `EDGE_BPF_PIN_DIR` | `/sys/fs/bpf/ad-event-processor` | Map pin path |
| `EDGE_XDP_MODE` / `XDP_MODE` | `generic` | Attach mode |
| `EDGE_XDP_REDIS_ADDRS` | `127.0.0.1:6479` | Entitlement + blocklist source (shard 0) |
| `REDIS_PASS` | from `.env` | Redis auth |
| `EDGE_BPF_SYNC_INTERVAL` / `SYNC_INTERVAL` | `5s` | Map sync tick |
| `EDGE_BPF_SYNC_METRICS_PORT` / `METRICS_PORT` | `9191` | Prometheus scrape |
| `XDP_SYN_COOKIE` | `0` | Enable SYN-cookie tail call |
| `XDP_FINGERPRINT` | `1` | Emit SYN fingerprint events |

Per-map overrides: `BPF_BLOCKLIST_MAP`, `BPF_ALLOWLIST_MAP`, etc. (see `cmd/edge-bpf-sync/doc.go`).

### CDN / L4 load balancer

When TCP terminates at CDN or cloud LB:

- Set `OS_FINGERPRINT_MISMATCH_ENABLED=false` and `TCP_SYN_SIG_ENABLED=false` on tracker, or accept skip metrics.
- XDP sees CDN edge IPs, not visitor IPs; blocklist value is limited to flood patterns on the edge link.
- JA3/TTL/MSS headers from XDP fingerprint pipeline are invalid behind CDN.

### Observability

| Endpoint / metric | Source |
| :--- | :--- |
| `GET :9191/metrics` | `edge-bpf-sync` |
| `ad_event_processor_edge_*` | nginx `/edge/metrics` (L7, independent) |
| `ad_edge_blocklist_lru_eviction_total` | bpf-sync on map pressure |
| Redis `xdp:stats` snapshot | Written every `STATS_INTERVAL` (2s) |

### Rollback

```bash
docker compose --profile enterprise-xdp stop edge-xdp
# or: stop edge-bpf-sync, then edge-xdp; optionally rm pin dir for clean reload
```

nginx L7 blacklist and tracker filters continue. No Postgres or outbox change required.

### Limitations

| Limit | Detail |
| :--- | :--- |
| Scope | L3/L4 only; no HTTP body or campaign logic |
| Evasion | Rotating proxies, CGNAT shared keys, CDN |
| LRU pressure | Kernel may evict cold host entries under flood; 5 min full resync reconciles shadow |
| LPM `BPF_F_NO_PREALLOC` | Bulk sync may allocate under load; monitor eviction metric |
| License | Unlicensed: maps pin but attach/sync idle |
| Hot path | Tracker must not import `internal/edge` |
| SLA | XDP does not use tracker `ad_http_request_duration_seconds` p99; layer-local only |

### Verification

```bash
go test ./internal/edge/ -short -run TestTokenBucket_ -count=1
go test ./internal/edge/ -short -run TestEdgeFilterRateLimitPerCPUHash_holdout -count=1
make gen bpf-dev
bash scripts/test/edge/lua_tests.sh compliance
sudo bash scripts/test/edge/xdp_resilience_drill.sh   # BTF + root; optional on CI runner
bash scripts/ci/compliance.sh
```

---

## Part 2: Multi-region

### Role

Regional spend and operation replication from edge cells to a global control Postgres. **Not** multi-master tracker ingest: `/track` stays on regional tracker pool; global control reconciles spend batches and operation leases.

Financial authority remains global Postgres (`current_spend`, `balance_ledger`). Regional WAL is a durability buffer, not ledger truth.

### Components

| Piece | Path | Role |
| :--- | :--- | :--- |
| `region-proxy` | `cmd/region-proxy`, `pkg/regionproxy/` | Regional WAL ingress, keygen, opkey pool, optional HTTP uplink |
| Regional `processor` | `cmd/processor` + `MULTI_REGION_ENABLED=1` | Flush spend sync to region-proxy broker topic |
| Global `control` | `cmd/control` | `POST /api/v1/region/ingest/batch`, operation leases, quorum |
| Quorum store | `pkg/regionproxy/quorum` | Redis leases `ad_event_processor:op:lease:{hex}` |

### Topology

```
Regional cell (AD_EVENT_PROCESSOR_REGION_CODE != 0)
  tracker / processor (local Redis, local PG optional replica)
  -> region-proxy (mmap WAL, gnet unix socket)
  -> uplink HTTP POST -> global control /api/v1/region/ingest/batch

Global cell (AD_EVENT_PROCESSOR_REGION_CODE = 0, MULTI_REGION_ENABLED=1)
  control + authoritative Postgres
  -> IngestRegionProxyBatch, dedup by factor_u + op_id
  -> operation leases when quorum required
```

Lab compose runs global control and regional `processor-1` + `region-proxy` on one host with `NET_MODE=host`. Production expects separate hosts per region within `max_regions` license cap.

### Features

| Feature | Detail |
| :--- | :--- |
| mmap WAL | 64 MiB segment; group commit; `Recover()` discards torn tail |
| Dedup keys | `keygen` stamps `FactorU` per record; global ingest rejects mismatch |
| Opkey pool | 16-byte operation IDs; watermark load shed when saturated |
| Quorum | `Book`/`Ack` on Redis before forward when replica count > 1 |
| Uplink | Batched HTTP to global; `X-Admin-API-Key` header; retry with forward unclaim |
| Spend sync | Processor `ProduceSpendSyncPayload` when `REGION_PROXY_ADDR` set |
| Slot migration | Requires `slot_migration` feature (Network+); separate from WAL path |

### Deploy (compose lab)

```bash
cp .env.example .env
# Apply JWT with multi_region: true (network or enterprise SKU).

# Global control (region code 0) - default stack
bash scripts/dev/stack/stack.sh full

# Regional lane
MULTI_REGION_ENABLED=1 \
AD_EVENT_PROCESSOR_REGION_CODE=1 \
GLOBAL_INGEST_URL=http://127.0.0.1:8188/api/v1/region/ingest/batch \
GLOBAL_INGEST_API_KEY="${ADMIN_API_KEY}" \
bash scripts/dev/stack/stack.sh multi-region up

# Optional: broker for processor ingress
bash scripts/dev/stack/stack.sh multi-region broker
```

Or:

```bash
docker compose -f deploy/compose/docker-compose.yaml --profile multi-region up -d region-proxy processor-1
```

Control boot fails `multi_region` without matching JWT feature (`internal/controlplane/serve.go`).

### Deploy (production outline)

1. **Global region** (`AD_EVENT_PROCESSOR_REGION_CODE=0`): control, Postgres primary, Redis shard 0 for quorum, ClickHouse if analytics required globally.
2. **Regional cells** (code 1..N): tracker pool, processor with `MULTI_REGION_ENABLED=1`, region-proxy, local Redis for WAL coordinator; point `GLOBAL_INGEST_URL` at global control HTTPS endpoint.
3. Set `REGION_PROXY_ADDR` on regional processor (unix socket or TCP per install).
4. Configure firewall: regional -> global `:8188` (or TLS ingress) only for uplink; no public exposure of region-proxy gnet socket.
5. Installer flag: `multi_region: true` in `deploy/installer/install.yaml` (see example).

### Environment

| Variable | Default | Component | Purpose |
| :--- | :--- | :--- | :--- |
| `MULTI_REGION_ENABLED` | `0` | control, processor | Enable multi-region mode |
| `AD_EVENT_PROCESSOR_REGION_CODE` | `0` | control, processor, region-proxy | `0` = global; non-zero = regional cell |
| `REGION_PROXY_ADDR` | `127.0.0.1:9093` | processor | Broker/gnet address for spend sync |
| `REGION_PROXY_REDIS_URL` | redis shard 0 | processor, region-proxy | Coordinator + ready probe |
| `GLOBAL_INGEST_URL` | `http://127.0.0.1:8188/api/v1/region/ingest/batch` | region-proxy | Uplink target |
| `GLOBAL_INGEST_API_KEY` | `ADMIN_API_KEY` | region-proxy | Auth header for uplink |
| `GLOBAL_SPEND_BATCH_MIN` | `100` | processor, control | Min batch before spend flush |
| `GLOBAL_SPEND_FLUSH_INTERVAL_MS` | `500` | control | Global spend aggregator tick |
| `GLOBAL_SPEND_MAX_CONCURRENCY` | `8` | control | Parallel global apply workers |

Regional processor **exits on boot** if `MULTI_REGION_ENABLED=1` and `REGION_PROXY_ADDR` is empty.

region-proxy flags (compose `command`): `-addr`, `-health-addr`, `-data-dir`, `-node-id`, `-region-code`, `-redis-url`, `-global-ingest-url`, `-global-ingest-api-key`.

### Import boundaries

| Rule | Detail |
| :--- | :--- |
| Hot ingest | `internal/ingest` must not import `pkg/regionproxy` |
| Tracker | `/track` must not block on uplink or quorum Redis |
| Processor | Wires region-proxy client only when `MULTI_REGION_ENABLED=1` |
| pkg | `pkg/regionproxy` must not import `internal/*` |

### Resilience targets (drill script)

| Target | Detail |
| :--- | :--- |
| Regional proxy failover RTO | < 120 s (operator checklist) |
| Budget invariant | `AssertBudgetInvariant` Redis vs PG +/-1 micro-unit after partition heal |
| Duplicate global apply | `proposal_rows=1` on replay |
| Fault proofs | >= 12 `mr_*` lines in controlplane fault tier |

```bash
bash scripts/test/multi_region_resilience_drill.sh
go test ./pkg/regionproxy/... -short -count=1
go test ./tests/e2e/ -run RegionProxy -count=1   # integration tier; Docker required
```

### Limitations

| Limit | Detail |
| :--- | :--- |
| SKU | Not on pilot, starter, pro, scale; `max_regions` caps cells (3 network, 99 enterprise) |
| Not appliance default | `single_vps` and pilot omit region-proxy |
| WAL size | Single 64 MiB segment per region-proxy data dir; plan disk and retention |
| Load shed | Opkey watermark returns backpressure; not unbounded queue |
| Global partition | Regional WAL retains records; uplink retries (`ForwardMaxAttempts` default 3) |
| CH / analytics | Regional ClickHouse is optional; reports may lag; not cross-region query fabric |
| Proof tier | Unit tests do not prove cutover; run fault drill + integration for wiring claims |

### Monitoring

| Signal | Meaning |
| :--- | :--- |
| `region_proxy_keygen_queue_depth` | WAL keygen backlog |
| `region_proxy_keygen_lag_p99_ms` | Keygen latency |
| region-proxy health socket | Ready after Redis probe |
| Global `ad_management_outbox_oldest_pending_seconds` | Unrelated but required healthy for config propagation |

---

## Combined production notes

| Concern | XDP | Multi-region |
| :--- | :--- | :--- |
| Host count | Counts toward `max_activations` | Each regional edge + global hosts count |
| Redis shard 0 | Blocklist + entitlement source | Quorum leases + changelog |
| Outbox | Pushes blacklist to shard 0 | Slot map, config still via global outbox |
| Release images | `edge-xdp` image in enterprise builds | `region-proxy` excluded from appliance image matrix |

Cross-ref: [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md), `.cursor/rules/edge.mdc`, `.cursor/rules/regions.mdc`, `.cursor/rules/xdp-bpf.mdc`.
