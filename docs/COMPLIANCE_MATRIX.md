# Compliance control matrix

Maps defensive-perimeter controls from `.cursor/rules/compliance.mdc` to implementation files and test proof. No HTMX UI; operator-facing architecture notes live in `docs/ARCHITECTURE.md`.

**CI gate:** `scripts/ci/compliance.sh` (invoked from `scripts/ci/local_check.sh`).

---

## Defensive measures (allowed)

| Control ID | Description | Implementation | Test / proof |
| :--- | :--- | :--- | :--- |
| CMP-DEF-01 | Wire-rate `XDP_DROP` on local NIC (blocklist, SYN/PPS limits) | `deploy/edge/xdp/bpf/edge_filter.c`, `cmd/edge-xdp`, `cmd/edge-bpf-sync` | `pkg/bpf/...` attach tests; `scripts/fault/` edge lab |
| CMP-DEF-02 | Passive TLS/TCP metadata (JA3/JA4 class); no covert device probes | `deploy/nginx/lua/edge-tls-hash.lua`, `internal/ingestion/device_filter.go`, `internal/management/service_fraud.go` (`TLSImpersonationWorker`) | `internal/ingestion/device_filter_test.go` |
| CMP-DEF-03 | In-line network tarpit (capped delay on own server; edge only) | `deploy/nginx/lua/edge-tarpit.lua`, `deploy/nginx/lua/access-check.lua` | `deploy/nginx/lua/tests/tarpit_test.lua`, `scripts/test/tarpit_test.sh` |
| CMP-EBPF-01 | BPF blocks only after local breach; TTL in map value | `deploy/edge/xdp/bpf/edge_filter.c`, `internal/edge/blocklist/` | `internal/edge/blocklist/*_test.go` |
| CMP-EBPF-02 | Immutable allowlist before deny in kernel | `allowlist.IsProtected`, `edge_filter.c` `allow_v4` | `internal/edge/blocklist/allowlist_test.go` |
| CMP-EBPF-03 | Sync path: management → Redis → `edge-bpf-sync` (no direct kernel writes from management) | `cmd/edge-bpf-sync`, management outbox `UPDATE_BLACKLIST` | `scripts/ci/compliance.sh` CMP-FORB-04 |
| M10-C3 | Fingerprint must not be sole L4 drop cause | `deploy/edge/xdp/bpf/edge_filter.c` | `scripts/ci/compliance.sh` M10-C3 |

---

## Forbidden measures (must not ship)

| Control ID | Description | Enforcement | Test / proof |
| :--- | :--- | :--- | :--- |
| CMP-FORB-01 | No DOM/Canvas/WebGL/audio fingerprint SDK | CI pattern scan | `scripts/ci/compliance.sh` |
| CMP-FORB-02 | No hack-back / reverse DDoS / flood helpers | CI pattern scan | `scripts/ci/compliance.sh` |
| CMP-FORB-03 | No integrated port scan / active probe of visitor hosts | CI pattern scan | `scripts/ci/compliance.sh` |
| CMP-FORB-04 | No `cilium/ebpf` import in management or tracker | `go list` import check | `scripts/ci/compliance.sh` |
| CMP-DEF-04 | No outbound connections to visitor/source IPs from management | CI pattern scan on management | `scripts/ci/compliance.sh` |

---

## Design and audit rules

| Control ID | Description | Implementation | Test / proof |
| :--- | :--- | :--- | :--- |
| CMP-AUDIT-01 | Blacklist mutations: PG txn + `admin_audit_log` + outbox | `internal/management/` blacklist handlers, outbox workers | `internal/management/*blacklist*_test.go` |
| CMP-AUDIT-02 | BPF mutations: `edge_block_audit` + bpf-sync logs | `cmd/edge-bpf-sync`, `internal/edge/` | Edge ops runbook |
| CMP-ALLOW-01 | `allowlist.IsProtected` before any deny persist/BPF sync | `internal/edge/blocklist/allowlist.go` | `allowlist_test.go` |
| CMP-PRIV-01 | Least privilege: only `edge-xdp` / `edge-bpf-sync` hold `CAP_BPF`/`CAP_NET_ADMIN` | Deploy manifests, `Dockerfile` entrypoints | Architecture review |
| M9-07 | No `SelectAndShard` in production tracker paths (StaticSlot only) | `internal/ingestion/static_slot_sharder.go` | `scripts/ci/compliance.sh` M9-07 |

---

## Observability (edge tarpit — GAP-CMP-01)

| Metric | Type | Source | Grafana |
| :--- | :--- | :--- | :--- |
| `espx_edge_tarpit_total` | counter | `deploy/nginx/lua/edge-metrics.lua` | `deploy/monitoring/grafana/provisioning/dashboards/edge.json` |
| `espx_edge_tarpit_delay_ms_total` | counter | `edge-metrics.lua` | `edge.json` |
| `ad_edge_tarpit_delay_seconds` | histogram | Tracker registry (reserved; edge emits ms counter) | N/A — edge path only |

**Profiles**

| Environment | `EDGE_TARPIT_ENABLED` | Config |
| :--- | :---: | :--- |
| Development / local | `false` (default) | `.env.example` |
| Production edge | `true` | `deploy/nginx/edge-production.env` |

**Env vars:** `EDGE_TARPIT_MAX_HEADERS` (64), `EDGE_TARPIT_BODY_BYTES` (65536), `EDGE_TARPIT_MAX_SEC` (2 dev, 10 prod; hard cap 15 in Lua).

---

## Data layer compliance (related)

| Control ID | Description | Implementation | Test / proof |
| :--- | :--- | :--- | :--- |
| CMP-PII-01 | PII hashed before ClickHouse insert | `pkg/piihash/`, `internal/ingestion/clickhouse_store.go` | `internal/ingestion/clickhouse_pii_test.go` |

Operator data security runbook (at-rest, TLS, secrets, retention): [runbooks/DATA_SECURITY.md](./runbooks/DATA_SECURITY.md).
| CMP-TELEM-01 | Vendor telemetry opt-in (`VENDOR_TELEMETRY_ENABLED`) | `pkg/vendorprobe/`, `internal/management/vendor_telemetry.go` | `pkg/vendorprobe/probe_test.go` |

---

## Fault proofs

| Fault | Script / test | Expected key |
| :--- | :--- | :--- |
| `edge_tarpit_triggered` | `scripts/test/tarpit_test.sh` with `EDGE_TARPIT_ENABLED=1` | `espx_edge_tarpit_total` increases |
| `edge_tarpit_triggered` | `deploy/nginx/lua/tests/tarpit_test.lua` | delay + metric increment (offline) |

---

## Change process

1. New defensive control: add row here + update `.cursor/rules/compliance.mdc` section 5 binding table.
2. New forbidden pattern: add CMP-FORB row + CI check in `scripts/ci/compliance.sh`.
3. PR checklist: `.cursor/rules/compliance.mdc` section 6.
