# Compliance - Defensive Perimeter and Forbidden Offense

eBPF binding: [EDGE.md](./EDGE.md#ebpf). Architecture: [../ARCHITECTURE.md](../ARCHITECTURE.md).

---

## 1. Defensive measures (allowed globally)

Defensive actions apply on customer-controlled infrastructure, use data already received by the server, and enforce access control at the on-prem perimeter. They do not target foreign systems.

### A. Wire-rate packet dropping (eBPF/XDP)

| Field | Detail |
| :--- | :--- |
| Action | `XDP_DROP` for malformed, abusive, or rate-exceeded traffic on the local NIC (`deploy/edge/xdp/bpf/edge_filter.c`: blocklist, SYN/PPS limits). |
| Art. 361 UK | Refusing to process packets on your interface; not altering a remote system. |
| US CFAA | Defending your own boundary; no unauthorized access to others' computers. |
| eSPX rule | Auto-blocks enter BPF maps only after local breach (rate limit, operator RBAC block, fraud outbox). TTL in map value; immutable allowlist before deny (`CMP-EBPF-*`). Sync: `management` -> Redis -> `cmd/edge-bpf-sync` -> pinned maps. No direct kernel writes from management. |

### B. Passive network signaling (JA3/JA4, TCP metadata)

| Field | Detail |
| :--- | :--- |
| Action | Parse protocol metadata the client sent to open a session: TLS ClientHello (JA3/JA4 class), TCP window/TTL where available, HTTP headers (`User-Agent`, `Sec-CH-UA*`). |
| GDPR / ePrivacy | No covert device probes; no non-disclosed hardware reads. Metadata is voluntarily transmitted for the handshake. |
| EU AI Act (2026) | Standard statistical anti-fraud: structural/machine signals, not prediction of personal human traits or biometric categorization. |
| eSPX rule | Edge/nginx exposes TLS hash -> `DeviceFilter` (`internal/ingestion/device_filter.go`): fraud signals; hard block only via operator list or policy. Cold path: `TLSImpersonationWorker` for JA3xUA mismatch analytics. No JS injection on publisher pages. |

### C. Network tarpitting (in-line slow response)

| Field | Detail |
| :--- | :--- |
| Action | Delay HTTP response to suspected bots (e.g. hold connection 5-10s) within HTTP semantics; resource pacing on your server. |
| Legal | Server may allocate its own CPU/socket budget; not an attack on client infrastructure. |
| eSPX status | Not implemented. If added: edge/nginx or gnet only; capped max delay (e.g. <= 15s); metric `edge_tarpit_delay_seconds`; never combined with outbound traffic to source IP. Not on billing/settlement paths. |

---

## 2. Offensive measures (forbidden)

Do not ship in code, docs, marketing, or operator runbooks as eSPX features.

### A. Active device fingerprinting and DOM probing

| Field | Detail |
| :--- | :--- |
| Action | Hidden JS: Canvas, WebGL, audio APIs, font enumeration, cross-site identity graph without consent. |
| GDPR / ePrivacy | Treated like illegal tracking; fines up to EUR20M or 4% global turnover. |
| EU AI Act | Prohibited / high-risk when used for covert behavioral or biometric categorization. |
| Art. 361 UK | Risk of unauthorized code execution on a foreign system (malware-like). |
| eSPX rule | Banned. CI `scripts/ci/check_compliance.sh` fails on fingerprint SDK patterns. Server-side AdTech only. |

### B. Active counter-attacks (hack back / strike back)

| Field | Detail |
| :--- | :--- |
| Action | Reverse DDoS, exploit payloads, flood-ping origin IPs detected in eBPF maps. |
| Art. 361 / 361-1 UK | Criminal unauthorized destructive network operations against external IPs. |
| US CFAA | Federal offense: intentional damage/transmission to external protected computers. |
| eSPX rule | `management`, `tracker`, `processor`, edge workers do not open outbound streams to source IPs being blocked or scored. Only local map/DB/redis mutations. No `SYN`/`UDP` flood helpers in repo. |

### C. Unsanctioned threat-intel scanning

| Field | Detail |
| :--- | :--- |
| Action | Auto `nmap`, reverse port scan, proxy probe, OS detection against visitor IPs. |
| Legal | Active scanning of foreign networks without authorization -> unauthorized access claims globally. |
| eSPX rule | No integrated port scanner. Geo/ASN from passive MaxMind DB only. IVT uses ClickHouse aggregates on data already ingested; no live probing of visitor hosts. |

---

## 3. Art. 361 UK - condensed risk map (Ukraine operators)

| Risk | Offensive (section 2) | Defensive (section 1) |
| :--- | :--- | :--- |
| R1 Identity / interference | 2.A DOM probing | 1.B passive TLS/headers |
| R2 Routing disruption | 2.B hack back; chaotic blocks | 1.A XDP + allowlist + audit |
| R3 Covert kernel change | Hidden BPF load | Declarative `edge-xdp` + install audit |

---

## 4. Design rules (engineering)

### 4.1 Single-writer and audit

- Blacklist mutations: PG txn + `admin_audit_log` + outbox `UPDATE_BLACKLIST` ([CONTROL_PLANE.md](./CONTROL_PLANE.md) section 1.3).
- BPF mutations: `edge_block_audit` + `edge-bpf-sync` logs.
- Every block: `reason`, `ttl`, `operator_id` or `rule_id`.

### 4.2 Immutable allowlist

Before any deny: `allowlist.IsProtected(ip)` - customer LAN, resolvers (`8.8.8.8/32`, `1.1.1.1/32`), loopback, declared gateways. Kernel: `allow_v4` checked before `blocklist_v4` in `edge_filter.c`.

### 4.3 Process privileges

| Binary | Root / caps | BPF | Outbound to visitor IPs |
| :--- | :--- | :--- | :--- |
| `management` | non-root | No | No (gRPC to internal services only) |
| `tracker` | tuned | No | No (responds to inbound only) |
| `edge-xdp`, `edge-bpf-sync` | `CAP_BPF`, `CAP_NET_ADMIN` | Yes | No |
| `ivt-detector`, `fraud-scorer` | service user | No | No scans; CH batch + management API |

### 4.4 Data flow (defensive path only)

Inbound packet/request -> optional XDP (local policy drop) -> nginx L7 (blacklist, rate limit) -> gnet tracker (passive headers/TLS hash, filters) -> accept or reject (optional future tarpit). Banned: outbound attack, JS probe, port scan.

---

## 5. Technical binding

| Defensive (section 1) | Component | Status |
| :--- | :--- | :--- |
| A XDP drop | `edge_filter.c`, `cmd/edge-xdp`, `cmd/edge-bpf-sync`, `allowlist.IsProtected` | Active |
| B TLS/JA3 class | `DeviceFilter`, nginx TLS hash header, `TLSImpersonationWorker` | Active |
| C Tarpit | nginx/gnet | Not implemented |
| Audit | `admin_audit_log`, `edge_block_audit` | Active |

Library: [`github.com/cilium/ebpf`](https://github.com/cilium/ebpf) - official map API only; no raw kernel memory.

---

## 6. PR / release checklist

- [ ] Change is section 1 defensive or neutral; not section 2 offensive.
- [ ] No new browser JS SDK, Canvas/WebGL, or cross-site identity graph.
- [ ] No outbound network to visitor/source IPs from management or workers.
- [ ] No `nmap`/port-scan dependencies or docs.
- [ ] Block path calls `allowlist.IsProtected` before persist/BPF sync.
- [ ] No `cilium/ebpf` import in `management` / `tracker`.
- [ ] `scripts/ci/check_compliance.sh` green.
- [ ] Update this guide if compliance surface changes.

---

## 7. Operator statement

Perimeter defense on customer-controlled infrastructure: drop abusive traffic, parse protocol metadata for fraud statistics, optional slow response to suspicious clients. No hack-back, hidden device fingerprinting, or visitor network scanning. Administrators authorize blocks and optional XDP; actions are auditable.

---

## 8. Vendor performance telemetry (opt-in)

Distinct from fraud passive telemetry (section 1.B) and license heartbeat (`ESPX_LICENSE_TELEMETRY` in [CONTROL_PLANE.md](./CONTROL_PLANE.md) section 8).

| Rule | Requirement |
| :--- | :--- |
| Default off | `telemetry_enabled: false` / `ESPX_VENDOR_TELEMETRY=0`; zero egress until operator enables |
| Cold path | `management` worker scrapes local `:9090` / `:metrics` only; POST once per interval |
| Anonymize | Strip campaign/customer/host labels; hostname -> `host_{uuid}`; no raw IPs or money metrics |
| Red zone | No user IPs, spend, CH rows, SQL, DSN; batch aborted if detected |
| Air-gap | Failed POST -> log + drop; no impact on `/track` |

Allowed telemetry: aggregated `MemStats`, gnet p50/p99, Redis Lua duration, CH batch/lag, XDP drop ratios. Banned: anything identifying end users or customer business.

Spec: [OPEN_GAPS.md](./OPEN_GAPS.md) GAP-ENG-03.

---

## 9. PII in ClickHouse (rolling hash)

Long-term CH storage uses hashed IP/UA (`ip_hash`, `ua_hash`) with daily rotating salt; not reversible without salt of that day. Raw PII stays in Redis for short fraud windows only; processor hashes on batch insert.

IVT/bot rules query `ip_hash` after cutover. Erasure bumps salt + CH mutation per `ErasureWorker`.

Spec: [OPEN_GAPS.md](./OPEN_GAPS.md) GAP-DATA-01.
