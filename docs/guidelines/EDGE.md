# Edge Ingress and eBPF

Nginx/OpenResty ingress and XDP L4 filter.

---

## Part I - Edge (Nginx/OpenResty)

## Listeners

| Port | Client | Upstream to tracker |
| :--- | :--- | :--- |
| `:8180` | HTTP/1.1 | HTTP/1.1 |
| `:443` | H2 / H3 / H1.1 (ALPN) | HTTP/1.1 |

TLS certs: `deploy/nginx/certs/generate-dev-certs.sh`. HTTP/3 needs nginx >= 1.25 with `ngx_http_v3_module`.

H2/H3: edge sets `X-Original-Method`, `X-Original-Path`, `X-TLS-Hash` (ClientHello MD5 class). Metric: `espx_edge_ingress_protocol_{h1,h2,h3}_total`.

---

## L7 pipeline (`access-check.lua`)

Phase 1 (pre-body):

| Check | Limit |
| :--- | :--- |
| Rate limit | 100 r/s baseline |
| Circuit breaker | Local |
| Blacklist | Cache refresh 5 s |
| Connections | 200 / IP, 8192 total |

Phase 2 (post-body): body DFA (`edge-parse-dfa.lua`), per-campaign RL (`edge-rl.lua`), proxy to tracker pool.

---

## Ingress schema

`TRACKER_INGRESS_SCHEMA` must match tracker `config.IngressSchema`:

| Schema | Edge field | Tracker parser |
| :--- | :--- | :--- |
| `openrtb_3` (production default) | `request.item[0].id` | `ParseOpenRTB3Ingress` |
| `espx_native` | `campaign_id` scan | `ParseTrackRequestJSON*` / vtproto |

---

## Shard selection

Must match Go `StaticSlotSharder`:

```text
slot  = crc32_castagnoli(campaign_id) & 1023
shard = slot_table[slot]
```

Lua: `edge-slot-map.lua`, `edge-shard-balancer.lua`. Links: [DATA_LAYER.md](./DATA_LAYER.md) Part I.

---

## Optional tarpit

When `EDGE_TARPIT_ENABLED=true`, `edge-tarpit.lua` slows or drops requests that exceed header count or body size thresholds before they reach the tracker. Env: `EDGE_TARPIT_MAX_HEADERS`, `EDGE_TARPIT_BODY_BYTES`, `EDGE_TARPIT_MAX_SEC` (see `.env.example`).

---

## Tracker wire parsers (optional)

| Path | Production use | Files |
| :--- | :--- | :--- |
| HTTP/1.1 FSM | Yes (edge -> H1.1 upstream) | `http1_fsm.go` |
| h2c on gnet | Evaluation only | `handler_http2.go`, `http2_*.go` |
| HTTP/3 sidecar | Evaluation only | `cmd/tracker-quic` |

Benchmarks: [HOT_PATH.md](./HOT_PATH.md).

---

## Part II - eBPF / XDP

Tiers A-C: LPM allow/deny blocklist, per-IP and global SYN limits, PPS token bucket, SYN cookies, /24 SYN cap, violation ringbuf to Redis autoban, passive TCP fingerprint to IVT pipeline (score only).

Sync path: management outbox -> Redis shard-0 -> `cmd/edge-bpf-sync` -> pinned maps at `/sys/fs/bpf/espx/`. Attach via `cmd/edge-xdp`.

Latency budget: XDP decision p99 < 10 us on ratelimit update path; negligible vs tracker SLA.

Compliance: defensive perimeter only; no outbound strike; fingerprint never sole `XDP_DROP` cause.

Key files: `deploy/edge/xdp/bpf/edge_filter.c`, `internal/edge/bpf/`, `internal/edge/blocklist/`, `cmd/edge-bpf-sync`.

Open: tier D (CO-RE portability, hardware offload), lab-only XDP chaos injector.

Links: [COMPLIANCE.md](./COMPLIANCE.md), [../ARCHITECTURE.md](../ARCHITECTURE.md).
