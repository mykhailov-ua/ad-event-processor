# Cold-path JSON ingress policy

Admin, payment, and control-plane HTTP APIs parse JSON with **stdlib** `encoding/json`, not the tracker’s custom DFA parser. This document is the canonical boundary companion to [PARSER_SECURITY.md](PARSER_SECURITY.md) §9.

**Scope:** `pkg/coldpath`, `internal/controlplane`, `internal/payment` webhooks, `internal/openrtb` admin validate — everything served from `cmd/control` / payment listeners, **not** gnet `:8181` ingest.

---

## 1. Canonical helpers

| Helper | File | Behavior |
| :--- | :--- | :--- |
| `DefaultMaxBody` | `pkg/coldpath/http.go` | **65536** bytes (64 KiB) |
| `SelfServePaymentIntentMaxBody` | same | **16384** bytes (16 KiB) |
| `PaymentWebhookMaxBody` | same | **65536** bytes (64 KiB) |
| `AlertmanagerWebhookMaxBody` | same | **1048576** bytes (1 MiB) |
| `RegionIngestMaxBody` | same | **4194304** bytes (4 MiB) |
| `ReadLimitedBody` | same | `http.MaxBytesReader` — oversize → 413 / read error |
| `DecodeBody[T]` | same | `json.Unmarshal` into typed struct |
| `DecodeRequestOrBadRequest` | same | read + decode; 400 on failure |

Handlers should use these helpers rather than raw `io.ReadAll(r.Body)`.

---

## 2. Per-route body limits

| Limit | Routes / module | Notes |
| ---: | :--- | :--- |
| **64 KiB** | Most `/api/v1/*` admin handlers | `coldpath.DefaultMaxBody` |
| **16 KiB** | `POST /api/v1/selfserve/payment-intents` | `coldpath.SelfServePaymentIntentMaxBody` |
| **64 KiB** | `POST /api/v1/consent` | `ops_handlers.go` (`ReadLimitedBody`) |
| **64 KiB** | Stripe / crypto payment webhooks | `coldpath.PaymentWebhookMaxBody` |
| **1 MiB** | `POST /ops/alertmanager/webhook` | `coldpath.AlertmanagerWebhookMaxBody` |
| **4 MiB** | `POST /api/v1/region/ingest/batch` | `coldpath.RegionIngestMaxBody` |
| **4 MiB** | Outbound edge metrics scrape | `edge_metrics_reader.go` (response, not ingress) |
| **8 MiB** | Outbound ops scrape readers | `ops_readers.go` (response, not ingress) |

Login body cap is regression-tested: `internal/controlplane/security_fixes_test.go` (`TestLoginBodySizeLimit`).

---

## 3. What hot-path parser hardening does **not** apply here

Tracker ingress budgets (**PS-G09–G13**, **PS-H02**, **PS-H05**) are **out of scope** for cold-path JSON:

| Hot-path control | Cold-path reality |
| :--- | :--- |
| `MaxJSONDepth` / `OrtbMaxJSONDepth` | No depth cap — stdlib recursion limits apply |
| `MaxJSONKeyPairs`, escape/WS scan budgets | Not enforced |
| `JSON_STRICT_UTF8` | Not enforced — stdlib UTF-8 rules |
| Custom DFA / zero-alloc reject | Allocations acceptable on admin path |
| `parser_chaos_drill` / fuzz nightly | No equivalent chaos corpus |

**Mitigations that do apply:** session/API-key auth, RBAC, per-route rate limits, `MaxBytesReader`, transactional outbox (handlers do not write Redis hot-path keys directly).

Admin OpenRTB validate (`POST …/validate-bid-request`) uses `internal/openrtb/validate.go` (`encoding/json` at 64 KiB), not `internal/ingestion` OpenRTB parsers.

---

## 4. Non-HTTP JSON paths

These read JSON from Postgres outbox, UDP control, or replica files — not public wire ingress:

| Area | File(s) | Notes |
| :--- | :--- | :--- |
| Outbox payloads | `controlplane/outbox.go` | Worker decodes queued jobs |
| Platform / lease | `service_platform.go`, `operation_lease.go` | Internal state |
| Telegram webhook | `tg_handlers.go` | 64 KiB wire limit; minimal `update_id` check; raw body may land in outbox |
| UDP control | `udp_control_server.go` | Operator channel, not browser-facing |

Treat outbox/replica JSON as **trusted-after-auth** data, not anonymous internet input.

---

## 5. Verification

```bash
# Dedicated CI gate (separate from parser security)
bash scripts/ci/cold_path_json_gate.sh

# Login 64 KiB cap
go test ./internal/controlplane/ -run=TestLoginBodySizeLimit -count=1

# Per-route oversize rejection
go test ./internal/controlplane/ -run=TestColdPathJSON -count=1
go test ./internal/controlplane/adminapi/ -run=TestColdPathJSON -count=1
go test ./internal/payment/ -run=TestColdPathJSON -count=1
go test ./pkg/coldpath/ -run=TestReadLimitedBody -count=1
```

There is **no** `fault_proof gap=PS-Gxx` line for cold-path JSON — parser security CI gates apply only to `internal/ingestion`.

---

## 6. Future hardening (backlog, not milestone PS-G)

If cold-path JSON needs algorithmic bounds (depth, key-pair count, strict UTF-8), treat as a **separate** workstream:

1. Shared `json.Decoder` wrapper in `pkg/coldpath` with `DisallowUnknownFields` where APIs are stable.
2. Optional depth limit via `Decoder.UseNumber` + custom token walk — only if abuse is observed on authenticated admin routes.
3. Do **not** import `internal/ingestion` parsers into controlplane (service boundary: `service-boundaries.mdc`).

Track backlog in `docs/DEVELOPMENT.md` only when a task is opened; do not duplicate here.

---

## 7. Related docs

| Doc | Topic |
| :--- | :--- |
| [PARSER_SECURITY.md](PARSER_SECURITY.md) §9 | Ingress parser scope boundary |
| [EDGE_CASES.md](EDGE_CASES.md) §9 | Parser vs infrastructure triage |
| [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md) | NIC-level drop (not JSON) |
| `.cursor/rules/cold-path.mdc` | Cold-path engineering rules |
