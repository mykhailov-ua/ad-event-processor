# Parser security and ingress hardening

Tracker parses every HTTP byte on the hot path before filters/billing. Scope: `internal/ingestion` + nginx edge seam. Related: [ARCHITECTURE.md](ARCHITECTURE.md), [DEVELOPMENT.md](DEVELOPMENT.md) §7.

## Design goals

1. Reject hostile wire early — zero allocs on reject path.
2. Close slow streams — incomplete bodies must not hold connections indefinitely.
3. Match edge — `/track` policy agrees with `edge-phase2.lua`.
4. Control cohort p99 &lt; **80 ms** under mixed chaos load; zero pool rejects.

Additive to tracker SLA (p95 &lt; 50 ms, p99 &lt; 80 ms).

## Edge ↔ tracker alignment

### `POST /track`

| Rule | Edge (Lua) | Tracker (gnet) |
| :--- | :---: | :---: |
| `Content-Length` required | yes | yes |
| Chunked rejected | yes | yes |
| Duplicate/obfuscated `TE` rejected | yes | yes |

Implementation: `http1TrackEdgePolicy` ↔ `deploy/nginx/lua/edge-phase2.lua`.

Verify: `go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1 -v` — `differential_count` must be **0**.

Chunked allowed on **`POST /openrtb/bid`** only (extensions rejected).

### `GET /click`

Query-only; `GET` only; strict UUID `campaign_id`. Gate: `EDGE_EXPOSE_CLICK` (404 when off on edge). Shard pick: CRC32 slot table.

## HTTP/1 incomplete-body (slow-stream DoS)

| Knob | Default | Behavior |
| :--- | :---: | :--- |
| `HTTP1_INCOMPLETE_MAX` | 3 | Close after N incomplete passes |
| `HTTP1_BODY_IDLE_MS` | 5000 | Wall-clock idle since body expected |
| `MaxRequestBodySize` | 1 MiB | Cap buffered incomplete body |

Metric: `ad_http1_incomplete_close_total{reason="spin|idle|buffer"}`. HTTP/2: `H2_INCOMPLETE_MAX=3`.

```bash
go test ./internal/ingestion/ -run=TestChaos_ParserSecurity_PS_G01 -count=1 -v
bash scripts/fault/parser_slow_body_drill.sh
```

Operator: set nginx `client_body_timeout` at perimeter.

## Framing and wire bombs

- **Chunk extensions:** `;` in chunk size lines rejected on all paths. Hex overflow guard; `chunkScratch` reset; cap drop above 64 KiB.
- **HTTP/2 header block:** 16 KiB cap (`h2MaxHeaderBlock`); excess → `RST_STREAM`.
- **TE.TE obfuscation:** duplicate `Transfer-Encoding`/`TE`, control chars, tab-obfuscated `chunked` rejected.
- **OpenRTB scan budget:**

| Env | Default | Effect |
| :--- | ---: | :--- |
| `ORTB_SCAN_MAX_BYTES` | 262144 | Top-level scan limit |
| `ORTB_MAX_QUOTE_CHECKS` | 65536 | Quote walk cap |

Metric: `ad_ortb_scan_truncated_total`.

- **Protobuf:** `PROTO_MAX_FIELDS=256` (default). Hot path: `unmarshalAdEventVT` with wire budget.
- **HPACK continuation:** 8 octet cap (`h2MaxIntContinuationOctets`).

## Sustained chaos load

Harness mixes ~90% valid protobuf `/track` + ~10% chaos. Pass: control p99 &lt; **80 ms**, `WorkerPoolRejectTotal` delta **0**, ≥500 control samples.

```bash
bash scripts/fault/parser_chaos_drill.sh
bash scripts/fault/parser_chaos_load.sh --duration=300s --rps=5000 --chaos-pct=10
```

| Env | Default |
| :--- | :--- |
| `CHAOS_LOAD_DURATION` | 300s / 8s (drill) |
| `CHAOS_LOAD_RPS` | 5000 |
| `CHAOS_LOAD_CHAOS_PCT` | 10 |
| `CHAOS_LOAD_P99_MS` | 80 |
| `CHAOS_LOAD_WORKERS` | 8 |

## Configuration

```bash
HTTP1_INCOMPLETE_MAX=3
HTTP1_BODY_IDLE_MS=5000
ORTB_SCAN_MAX_BYTES=262144
ORTB_MAX_QUOTE_CHECKS=65536
PROTO_MAX_FIELDS=256
H2_INCOMPLETE_MAX=3
```

## JSON parse budgets

| Constant | Default | Effect |
| :--- | ---: | :--- |
| `MaxJSONTotalWSkip` | 4096 | Cumulative whitespace |
| `MaxJSONStringScanBytes` | 65536 | Bytes in one string |
| `MaxJSONStringEscapes` | 16384 | Escapes per string |
| `MaxJSONKeyPairs` | 10000 | Key:value pairs |
| `JSON_STRICT_UTF8` | on | Reject ill-formed UTF-8 |

Depth: `MaxJSONDepth=16` (track), `OrtbMaxJSONDepth=32` (ORTB ingress).

## Verification checklist

| Step | Command |
| :--- | :--- |
| Parser chaos | `go test ./internal/ingestion/ -run=TestChaos_ParserSecurity -count=1 -v` |
| Cross-hop | `go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1` |
| TE/proto/HPACK | `go test ./internal/ingestion/ -run='TestChaos_TE_TE\|TestChaos_Proto_FieldBudget\|TestChaos_HPACK' -count=1` |
| Sustained load | `bash scripts/fault/parser_chaos_load.sh --duration=8s --rps=3000` |
| Full drill | `bash scripts/fault/parser_chaos_drill.sh` |
| Alloc gates | `bash scripts/test/gate_bench.sh` |

Chaos tests emit `fault_proof gap=open|closed`.

## Out of scope

| Boundary | In scope | Out of scope |
| :--- | :--- | :--- |
| Packages | `internal/ingestion`, nginx lua | `pkg/coldpath`, controlplane, payment |
| Admin JSON | — | `encoding/json` + `coldpath.DefaultMaxBody` (64 KiB) |
| ML/fraud | Redis boost snapshots | `internal/fraud`, `cmd/fraud-scorer` |
| XDP | — | [XDP.md](XDP.md) |
| TCP/OS | HTTP/1 incomplete close | `scripts/ops/sysctl.sh`, SYN drop gates |

Cold-path gate: `bash scripts/ci/cold_path_json_gate.sh`. When p99 spikes but parser proofs pass, check Redis RTT, accept backlog, cgroup throttle first.
