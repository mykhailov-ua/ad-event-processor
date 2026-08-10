# Parser security and ingress hardening

BidShard’s tracker parses every byte of HTTP traffic on the hot path before filters or billing run. Parser bugs and parser differentials are a common source of denial-of-service and request-smuggling incidents in 2026 advisories. This document describes what the tracker enforces, how that lines up with the nginx edge, and how to verify it in CI and drills.

**Scope:** `internal/ingestion` — HTTP/1–2 wire parsers, JSON/OpenRTB/protobuf body parsers, and the nginx ↔ gnet ingress seam.

**Related:** [ARCHITECTURE.md](ARCHITECTURE.md) (request lifecycle), [DEVELOPMENT.md](DEVELOPMENT.md) §7 (fault drills), [BENCHMARKS.md](BENCHMARKS.md) (micro-bench gates), [EDGE_CASES.md](EDGE_CASES.md) §9 (parser vs OS/network failures), [COLD_PATH_JSON.md](COLD_PATH_JSON.md) (admin JSON — out of scope).

Engineering backlog: `.cursor/PARSER_SECURITY_MILESTONE.md` (phases **P0–P3**, gaps **PS-G01–G13**, **PS-H01–H06** — all closed in code).

---

## 1. Design goals

1. **Reject hostile wire early** — malformed framing, smuggling tokens, and algorithmic bombs should fail in microseconds with zero heap allocations on the reject path.
2. **Close slow streams** — incomplete bodies must not hold connections or worker slots indefinitely (Slow JSON Stream class).
3. **Match the edge** — `POST /track` policy on the tracker must agree with nginx `edge-phase2.lua` so attackers cannot pick a weaker hop.
4. **Protect the control cohort** — under mixed valid + chaos load, handler p99 for legitimate traffic stays below **80 ms** and the worker pool must not reject work.

These gates are **additive** to the tracker SLA in `espx.mdc` (p95 &lt; 50 ms, p99 &lt; 80 ms).

---

## 2. Edge ↔ tracker alignment (`POST /track`)

The appliance nginx perimeter applies a strict wire policy before traffic reaches gnet on port 8181:

| Rule | Edge (Lua) | Tracker (gnet) |
| :--- | :---: | :---: |
| `Content-Length` required | yes | yes |
| `Transfer-Encoding: chunked` rejected | yes | yes |
| Duplicate / obfuscated `TE` headers rejected | yes | yes |

Implementation: `http1TrackEdgePolicy` in `handler_http1_ingress_canonical.go`, mirrored from `deploy/nginx/lua/edge-phase2.lua`.

**Why it matters:** Chunked encoding and missing `Content-Length` on `/track` are valid HTTP elsewhere but are disallowed on this path so body boundaries are unambiguous. Direct access to `:8181` in dev must follow the same rules as traffic through nginx.

**Verification:** `TestChaos_CrossHop_NginxGnet` runs a 237-vector corpus; `differential_count` must be **0**.

```bash
go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1 -v
```

Chunked transfer encoding remains supported on **`POST /openrtb/bid`** (exchange path), with chunk extensions rejected (see §4).

---

## 3. HTTP/1 incomplete-body policy (slow-stream DoS)

**Threat:** An attacker sends a valid request line and headers, then drips body bytes slowly. A parser that waits forever ties up gnet connections and worker pool slots.

**Mitigation:**

| Knob | Default (prod) | Dev (`.env.example`) | Behavior |
| :--- | :---: | :---: | :--- |
| `HTTP1_INCOMPLETE_MAX` | 3 | 3 | Close after N `OnTraffic` passes that end in `errIncompleteRequest` with no forward progress |
| `HTTP1_BODY_IDLE_MS` | 5000 | 5000 | Wall-clock idle since body was expected; close if exceeded |
| `MaxRequestBodySize` | 1 MiB | 1 MiB | Cap buffered incomplete body in connection scratch |

**Metric:** `ad_http1_incomplete_close_total{reason="spin|idle|buffer"}`.

HTTP/2 already had `H2_INCOMPLETE_MAX` (default 3); HTTP/1 now mirrors that pattern.

**Verification:**

```bash
# Unit proof (PS-G01)
go test ./internal/ingestion/ -run=TestChaos_ParserSecurity_PS_G01 -count=1 -v

# Slow-body drill
bash scripts/fault/parser_slow_body_drill.sh
```

**Operator note:** At nginx, set `client_body_timeout` and minimum upload rate so slow clients are dropped at the perimeter before they reach gnet. Document values in your deploy runbook.

---

## 4. Framing and wire bombs

### Chunk extensions (HTTP/1)

Chunk size lines containing `;` (RFC chunk extensions) are **rejected** on all paths. This closes CRLF-in-extension parser differentials (Netty CVE-2026-33870 class).

Hex chunk sizes are parsed with an overflow guard before each `size*16` step; absurd size lines (e.g. 16+ hex digits) are rejected with `errInvalidRequest` before body slicing.

Per-connection `chunkScratch` for fragmented chunked bodies is reset after each completed HTTP/1 parse and on connection close; capacity above **64 KiB** is dropped (PS-H03).

### HTTP/2 header block cap

Assembled HPACK header blocks per request are capped at **16 KiB** (`h2MaxHeaderBlock` in `http2_conn.go`). Continuation frames that would exceed the cap trigger `RST_STREAM` / connection teardown (PS-G07 companion).

### TE.TE obfuscation

Duplicate `Transfer-Encoding` / `TE` headers and values with control characters or tab-obfuscated `chunked` tokens are rejected.

### OpenRTB 2.6 scan budget

Quote-dense or oversized JSON can force unbounded scanning. Caps:

| Env | Default | Effect |
| :--- | ---: | :--- |
| `ORTB_SCAN_MAX_BYTES` | 262144 | Stop top-level scan after this many bytes |
| `ORTB_MAX_QUOTE_CHECKS` | 65536 | Cap quote-character walks |

**Metric:** `ad_ortb_scan_truncated_total`.

Oversized payloads without a valid `imp` prefix are fast-rejected before deep scanning.

### Protobuf wire field budget

| Env | Default | Effect |
| :--- | ---: | :--- |
| `PROTO_MAX_FIELDS` | 256 | Reject vtproto wire with too many fields (field-flood class) |

Hot path uses `unmarshalAdEventVT` with a wire scan budget (`proto_wire_budget.go`), not unbounded `UnmarshalVT`.

### HPACK continuation cap

HTTP/2 integer continuation sequences are capped at **8 octets** (`h2MaxIntContinuationOctets` in `http2_varint.go`) to block integer bomb advisories (CVE-2026-59248 class).

---

## 5. Sustained chaos load (mixed traffic)

**Threat:** Under production RPS, a fraction of hostile requests must not degrade legitimate traffic or exhaust the worker pool.

The load harness (`parser_chaos_load.go`) mixes roughly **90%** valid protobuf `POST /track` with **10%** chaos (whitespace JSON bomb, quote-dense OpenRTB, fragmented chunked ORTB, slow-body prefix, proto field flood, malformed wire).

**Pass criteria:**

| Check | Gate |
| :--- | :--- |
| Control cohort p99 | &lt; **80 ms** (`CHAOS_LOAD_P99_MS`, default 80) |
| `WorkerPoolRejectTotal` delta | **0** |
| Control samples | ≥ 500 per run |

**Drill scripts:**

```bash
# CI / nightly drill (8 s smoke inside full parser drill)
bash scripts/fault/parser_chaos_drill.sh

# Manual / pre-release (5 min @ 5k RPS target)
bash scripts/fault/parser_chaos_load.sh --duration=300s --rps=5000 --chaos-pct=10
```

**Load harness env (optional overrides):**

| Env | Default | Purpose |
| :--- | :--- | :--- |
| `CHAOS_LOAD_DURATION` | 300s (script) / 8s (drill) | Run length |
| `CHAOS_LOAD_RPS` | 5000 | Target dispatch rate |
| `CHAOS_LOAD_CHAOS_PCT` | 10 | Hostile fraction (percent) |
| `CHAOS_LOAD_P99_MS` | 80 | p99 budget for control cohort |
| `CHAOS_LOAD_WORKERS` | 8 | Parallel `OnTraffic` workers |

The in-process harness measures handler latency and pool health, not line-rate fidelity to real network RPS. Achieved RPS in lab is typically lower than the target; the p99 and pool-reject gates are authoritative.

Successful runs log:

```text
fault_proof fault=parser_chaos_load gap_id=PS-G08 gap=closed pool_rejects=0 p99_ms=...
```

---

## 6. Configuration reference

Production defaults are set in `internal/config/env.go`; development overrides are in `.env.example`:

```bash
HTTP1_INCOMPLETE_MAX=3
HTTP1_BODY_IDLE_MS=5000
ORTB_SCAN_MAX_BYTES=262144
ORTB_MAX_QUOTE_CHECKS=65536
PROTO_MAX_FIELDS=256
H2_INCOMPLETE_MAX=3          # HTTP/2 incomplete frames (pre-existing)
```

---

## 7. Verification checklist

| Step | Command |
| :--- | :--- |
| All PS-G proof stubs | `go test ./internal/ingestion/ -run=TestChaos_ParserSecurity -count=1 -v` |
| Cross-hop corpus | `go test ./internal/ingestion/ -run=TestChaos_CrossHop_NginxGnet -count=1` |
| Phase P2 TE / proto / HPACK | `go test ./internal/ingestion/ -run='TestChaos_TE_TE|TestChaos_Proto_FieldBudget|TestChaos_HPACK' -count=1` |
| Sustained load | `bash scripts/fault/parser_chaos_load.sh --duration=8s --rps=3000` |
| JSON hardening (G09–G13) | `go test ./internal/ingestion/ -run='TestChaos_ParserSecurity_PS_G09|TestChaos_ParserSecurity_PS_G1[0-3]' -count=1` |
| Full parser drill | `bash scripts/fault/parser_chaos_drill.sh` |
| Alloc + micro benches | `bash scripts/test/gate_bench.sh` |

Every chaos test emits a `fault_proof` line (`gap=open|closed`) for CI grep gates.

---

## 8. Gap summary (phases P0–P3)

| Phase | ID | Topic | Status |
| :--- | :--- | :--- | :---: |
| P0 | PS-G01 | HTTP/1 slow-body / incomplete hold | closed |
| P1 | PS-G02 | Chunk-extension framing | closed |
| P1 | PS-G03 | OpenRTB quote-dense scan CPU | closed |
| P1 | PS-G04 | Edge ↔ tracker differential | closed |
| P1 | PS-H01 | `requestBufferPool` cap poisoning | closed |
| P2 | PS-G05 | TE.TE obfuscation | closed |
| P2 | PS-G06 | Protobuf wire field budget | closed |
| P2 | PS-G07 | HPACK continuation cap | closed |
| P2 | PS-G08 | Sustained chaos load mix | closed |
| P2 | PS-G09 | Unicode homoglyph / non-ASCII keys | closed |
| P2 | PS-G10 | Duplicate JSON key last-wins | closed |
| P2 | PS-G11 | Lone Unicode surrogate in strings | closed |
| P2 | PS-G12 | Distributed whitespace bomb | closed |
| P2 | PS-G13 | Quote-dense / escape-flood strings | closed |
| P2 | PS-H02 | Per-object JSON key-pair flood (`MaxJSONKeyPairs`) | closed |
| P2 | PS-H03 | `chunkScratch` per-connection retention | closed |
| P2 | PS-H06 | Extended fuzz + nightly 2 h (smoke in drill) | closed |
| P3 | PS-H04 | ORTB3 key-scan escape walk | closed |
| P3 | PS-H05 | Strict UTF-8 in JSON string values | closed |

DoD detail: `.cursor/PARSER_SECURITY_MILESTONE.md` §6 (PS-H), §1 (PS-G).

### JSON parse budgets (PS-G09–G13, PS-H02)

| Constant | Default | Effect |
| :--- | ---: | :--- |
| `MaxJSONTotalWSkip` | 4096 | Cumulative whitespace per JSON document |
| `MaxJSONStringScanBytes` | 65536 | Bytes examined in one string value |
| `MaxJSONStringEscapes` | 16384 | Backslash escapes per string value |
| `MaxJSONKeyPairs` | 10000 | Key:value pairs per JSON document (track + ORTB3) |
| `JSON_STRICT_UTF8` | on (`0` disables) | Reject ill-formed / overlong UTF-8 in string values |

Track JSON requires literal ASCII keys; `campaign_id` must be a literal UUID string. OpenRTB top-level section keys use last-wins at root depth only (nested `"source"` in `eids` cannot shadow the top-level `source` object).

Nested JSON values (track `payload` field and OpenRTB skip paths) use `skipJSONValueBudgetDepth` with `scanJSONStringEnd` inside `{`/`[` so escape bombs cannot bypass the string budget. Depth caps: `MaxJSONDepth` (16) for track, `OrtbMaxJSONDepth` (32) for OpenRTB ingress.

---

## 9. What this does not cover

Phases **P0–P3** (**PS-G01–G13**, **PS-H01–H06**) close **tracker ingress** only (`internal/ingestion` + nginx edge seam). The table below is the explicit boundary — do not file hot-path parser regressions against these layers.

| Boundary | In scope ([§1–§8](#1-design-goals)) | Out of scope (this milestone) |
| :--- | :--- | :--- |
| **Packages** | `internal/ingestion`, `deploy/nginx/lua` edge DFA | `pkg/coldpath`, `internal/controlplane`, `internal/payment`, `internal/openrtb` validate |
| **Wire / parser** | Custom DFA, vtproto wire budget, HPACK/TE guards | Stdlib `encoding/json` on admin APIs |
| **Body limits** | `MaxRequestBodySize`, ORTB scan, `PROTO_MAX_FIELDS` | `coldpath.DefaultMaxBody` (64 KiB) + per-route overrides — [COLD_PATH_JSON.md](COLD_PATH_JSON.md) |
| **Verification** | `TestChaos_ParserSecurity`, `parser_chaos_drill.sh`, fuzz nightly | Admin auth + `MaxBytesReader`; no PS-G chaos corpus |
| **ML / fraud scoring** | Tracker reads Redis `ml:score:boost:*` snapshots only | `internal/fraud`, `cmd/fraud-scorer` — model training and batch scoring |
| **NIC / XDP** | — | Optional Enterprise kernel drop — [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md) |
| **TCP / OS** | HTTP/1 incomplete close on gnet | Listen backlog, netem RTO, cgroup throttle — [EDGE_CASES.md](EDGE_CASES.md) §9 |

### 9.1 Cold-path JSON

Admin (`:8188`), payment webhooks, and control-plane batch ingest use **`pkg/coldpath`** with **`encoding/json`**. They do **not** inherit `MaxJSONKeyPairs`, `JSON_STRICT_UTF8`, or fuzz/chaos CI from the tracker parser milestone.

**Canonical doc:** [COLD_PATH_JSON.md](COLD_PATH_JSON.md) (limits table, helpers, verification).

### 9.2 ML / fraud scoring

The tracker hot path must **not** import `internal/fraud`. Fraud scores arrive asynchronously via Redis; scoring logic lives in `cmd/fraud-scorer` / processor microbatch. Parser security does not cover model inference, ONNX paths, or feature extraction.

### 9.3 NIC-level XDP

Enterprise optional: BPF blacklist drop on the ingress NIC **before** userspace. Default appliance uses nginx Lua (`access_check.lua`). XDP does not parse HTTP or JSON; it is perimeter volume control, not a parser substitute.

**Doc:** [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md). Compliance fingerprint: `bash scripts/ci/compliance.sh`.

### 9.4 TCP listen overflow / netem

`ListenOverflows`, SYN backlog, 1% lo loss, and Redis RTT under netem can raise handler p99 without any parser change. Fixing `somaxconn`, edge connection churn, and cgroup CPU is **ops / infrastructure** work.

**Triage:** [EDGE_CASES.md](EDGE_CASES.md) §9 symptom table. **Drills:** `scripts/perf/purgatory/`, `scripts/test/tcp_syn_drop_gate.sh`.

When handler p99 spikes but `gate_bench.sh`, `parser_chaos_drill.sh`, and PS-G proofs pass, investigate Redis RTT, accept backlog, and cgroup throttle **before** opening a parser regression.
