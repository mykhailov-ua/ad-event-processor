# Residential proxy and L7 desync detection backlog

**Status: CLOSED** (all slugs shipped as of 2026-08-27)

OSI-layer mismatch detection between edge TLS/TCP signals and HTTP application semantics. Baseline audit: prior gap analysis vs `internal/ingestion`, `internal/edge`, `deploy/nginx/lua/`.

**Canonical behavior:** `deploy/vendor/ANTIFRAUD.md` (signals include `h2_*`, `sec_fetch_anomaly`, `client_hints_mismatch`, `tls_alpn_mismatch`, `device_mismatch`, `os_fingerprint_mismatch`, `tcp_mss_anomaly`, `residential_proxy`, `layer_desync_count`).

**Out of scope here:** ML feature expansion (`cmd/fraud-scorer`), admin UI rebuild (`web/` removed), migration parity (`competitive_backlog.md`).

Cross-reference slugs in PR descriptions for historical context only.

---

## Placement rules

| Tier | Allowed | Forbidden |
| :--- | :--- | :--- |
| Edge (nginx Lua, XDP) | JA3/JA4, ALPN from ClientHello, TCP SYN capture, H2 wire before HPACK flatten | Postgres, ClickHouse, per-request external HTTP |
| Hot (`internal/ingestion`) | O(1) header scans, snapshot tables, pre-bound metrics | `fmt.Sprintf`, dynamic Prom labels, `internal/fraud` scoring import |
| Cold (`cmd/ivt-detector`, processor) | CH aggregates, batch rules, feed file append | Sync call from `/track` request thread |

CDN / ALB without `X-TCP-*` and edge TLS headers: signals must **fail-open** (documented in `ANTIFRAUD.md` § CDN and TLS fingerprint limits).

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [x] Every new symbol resolves (`go build -o /dev/null ./cmd/tracker/` exit 0, 2026-08-27)
- [x] Hot path does not import `internal/fraud` scoring (`boundaries.mdc`, `hot-path.mdc`; no `internal/fraud` import in `internal/ingestion/*.go` production sources)
- [x] Verification commands pasted below with package path (`quality.mdc`)
- [x] Holdout or fault test when behavior is non-obvious (`testing.mdc`; `*_holdout*` in `h2_fingerprint_test.go`, `l7_wire_filter_test.go`, `layer_desync_test.go`, etc.)
- [x] No microbench cited as prod `/track` p99 (`core.mdc`, `anti-slop.mdc`)
- [x] `bash scripts/ci/check_no_legacy_naming.sh` clean (exit 0, 2026-08-27)
- [x] CDN fail-open path documented when signal needs `X-TCP-*` / `X-TLS-*` (`ANTIFRAUD.md` § CDN and TLS fingerprint limits; per-filter absent-header skips)

---

## Summary table

| Slug | Priority | Tier | Status |
| :--- | :--- | :--- | :--- |
| `sec_fetch_metadata_validate` | P1 | hot | **shipped** (`L7WireFilter`, signal `sec_fetch_anomaly`) |
| `client_hints_platform_matrix` | P1 | hot | **shipped** (`L7WireFilter`, signal `client_hints_mismatch`) |
| `tls_alpn_browser_mismatch` | P1 | edge + hot | **shipped** (edge `X-TLS-ALPN`, signal `tls_alpn_mismatch`) |
| `h2_settings_frame_fingerprint` | P1 | edge + hot | **shipped** (tracker `parseH2Ingress`, `h2_settings_mismatch`) |
| `h2_pseudo_header_order` | P1 | edge + hot | **shipped** (`h2_pseudo_order_mismatch`) |
| `h2_h1_downgrade_artifact` | P2 | hot | **shipped** (`h2_downgrade_artifact`) |
| `http1_header_order_fingerprint` | P1 | hot | **shipped** (`header_order_mismatch`) |
| `accept_encoding_browser_consistency` | P2 | hot | **shipped** (`accept_encoding_mismatch`) |
| `accept_language_geo_hot_signal` | P2 | hot | **shipped** (`accept_lang_geo_mismatch`) |
| `tls_ja4_browser_corpus` | P1 | edge + hot | **shipped** (`tls_ja4_mismatch`) |
| `tcp_mss_asn_tunnel_heuristic` | P2 | hot | **shipped** (`tcp_tunnel_mss`) |
| `tcp_p0f_signature_correlation` | P2 | edge | **shipped** (`tcp_syn_os_mismatch`) |
| `track_json_serialization_fingerprint` | P2 | hot | **shipped** (`json_serialization_bot`) |
| `track_behavior_telemetry_ingest` | P2 | hot + client | **shipped** (`behavior_telemetry_missing`, `track_telemetry.js`) |
| `rtt_split_tunnel_detect` | P3 | edge + cold | **shipped** (cold `ivt-detector` rule; CH `rtt_split_delta_ms`) |
| `residential_intel_hot_path_read` | P1 | hot | **shipped** (`residential_intel_table.go` hot snapshot) |
| `cross_layer_fraud_score_snapshot` | P2 | hot | **shipped** (`layer_desync_count` on fraud stream) |

Existing slug (unchanged scope): `mobile_biometrics_pipeline` in [competitive_backlog.md](./competitive_backlog.md) — **done** (gyro/touch ingest + cold `ivt-detector` rule).

### Shipped env knobs (tracker)

| Env | Default | Signal |
| :--- | :--- | :--- |
| `SEC_FETCH_VALIDATE_ENABLED` | `true` | `sec_fetch_anomaly` |
| `CLIENT_HINTS_PLATFORM_ENABLED` | `true` | `client_hints_mismatch` |
| `TLS_ALPN_MISMATCH_ENABLED` | `true` | `tls_alpn_mismatch` |
| `H2_SETTINGS_FINGERPRINT_ENABLED` | `true` | `h2_settings_mismatch` |
| `H2_PSEUDO_ORDER_ENABLED` | `true` | `h2_pseudo_order_mismatch` |
| `H2_DOWNGRADE_ARTIFACT_ENABLED` | `true` | `h2_downgrade_artifact` |
| `HTTP1_HEADER_ORDER_ENABLED` | `true` | `header_order_mismatch` |
| `ACCEPT_ENCODING_BROWSER_ENABLED` | `true` | `accept_encoding_mismatch` |
| `ACCEPT_LANG_GEO_ENABLED` | `false` | `accept_lang_geo_mismatch` (requires campaign `accept_lang_geo_enabled`) |
| `TLS_JA4_BROWSER_CORPUS_ENABLED` | `true` | `tls_ja4_mismatch` |
| `TCP_MSS_TUNNEL_ENABLED` | `true` | `tcp_tunnel_mss` (requires `X-TCP-MSS` + GeoIP ASN) |
| `TCP_MSS_TUNNEL_THRESHOLD` | `1400` | MSS compare threshold for `tcp_tunnel_mss` |
| `TCP_SYN_SIG_ENABLED` | `true` | `tcp_syn_os_mismatch` (requires `X-TCP-SIG` from edge) |
| `JSON_SERIALIZATION_FINGERPRINT_ENABLED` | `false` | `json_serialization_bot` (requires campaign `json_serialization_enabled`) |

---

## Shipped slug index

| Slug | Priority | Tier | Status |
| :--- | :--- | :--- | :--- |
| `sec_fetch_metadata_validate` | P1 | hot | **shipped** (`L7WireFilter`, `sec_fetch_anomaly`) |
| `client_hints_platform_matrix` | P1 | hot | **shipped** (`L7WireFilter`, `client_hints_mismatch`) |
| `tls_alpn_browser_mismatch` | P1 | edge + hot | **shipped** (`X-TLS-ALPN`, `tls_alpn_mismatch`) |
| `h2_settings_frame_fingerprint` | P1 | edge + hot | **shipped** (`parseH2Ingress`, `h2_settings_mismatch`) |
| `h2_pseudo_header_order` | P1 | edge + hot | **shipped** (`h2_pseudo_order_mismatch`) |
| `h2_h1_downgrade_artifact` | P2 | hot | **shipped** (`h2_downgrade_artifact`) |
| `http1_header_order_fingerprint` | P1 | hot | **shipped** (`header_order_mismatch`) |
| `accept_encoding_browser_consistency` | P2 | hot | **shipped** (`accept_encoding_mismatch`) |
| `accept_language_geo_hot_signal` | P2 | hot | **shipped** (`accept_lang_geo_mismatch`) |
| `tls_ja4_browser_corpus` | P1 | edge + hot | **shipped** (`tls_ja4_mismatch`) |
| `tcp_mss_asn_tunnel_heuristic` | P2 | hot | **shipped** (`tcp_tunnel_mss`, ASN + MSS threshold) |
| `tcp_p0f_signature_correlation` | P2 | edge | **shipped** (`tcp_syn_os_mismatch`, `X-TCP-SIG`) |
| `track_json_serialization_fingerprint` | P2 | hot | **shipped** (`json_serialization_bot`, byte scan) |
| `track_behavior_telemetry_ingest` | P2 | hot + client | **shipped** (`behavior_telemetry_missing`, `track_telemetry.js`) |
| `rtt_split_tunnel_detect` | P3 | edge + cold | **shipped** (cold `ivt-detector`; no hot signal) |
| `residential_intel_hot_path_read` | P1 | hot | **shipped** (Redis snapshot hot read) |
| `cross_layer_fraud_score_snapshot` | P2 | hot | **shipped** (`layer_desync_count` + `GET /api/v1/reports/layer-desync-summary`) |

---

## `h2_settings_frame_fingerprint`

**Priority:** P1

**Gap:** `parseH2Ingress` acks client SETTINGS but does not record parameter order, values (`HEADER_TABLE_SIZE`, `ENABLE_PUSH`, `INITIAL_WINDOW_SIZE`, …), or post-SETTINGS `WINDOW_UPDATE` increment. Stealth libraries diverge from Chrome (e.g. `ENABLE_PUSH: 0`, `INITIAL_WINDOW_SIZE: 6291456`, window increment `15663105`).

**Surface:** `deploy/nginx/lua/` (terminate TLS, see raw frames) or tracker `internal/ingestion/http2_conn.go` + new `X-H2-Settings` / `X-H2-Window-Update` edge headers.

**Target:**

- Capture canonical SETTINGS tuple + first WINDOW_UPDATE per connection (or first stream).
- Snapshot table: Chrome / Firefox / Safari / known-bad (httpx, Go `net/http`, curl-impersonate gaps).
- New L2-weak signal `h2_settings_mismatch` (weight TBD in `fraudReasonRegistry`); fail-open when header absent.

### Done gates

- [x] Corpus fixtures in `h2_fingerprint_test.go`; holdout: Chrome tuple passes, httpx/Go defaults fail (`TestH2SettingsAnomaly_holdout*`, `TestH2Ingress_captureChromeFingerprint`)
- [x] Metric via `ad_fraud_reason_total{reason="h2_settings_mismatch"}` (no dedicated `ad_h2_settings_mismatch_total`; `internal/metrics/collectors.go`)
- [x] `ANTIFRAUD.md` signal row + CDN fail-open note

---

## `h2_pseudo_header_order`

**Priority:** P1

**Gap:** HPACK decode loses pseudo-header order. Chrome `:method,:authority,:scheme,:path` vs Firefox `:method,:path,:authority,:scheme` not distinguished; bots often reorder.

**Surface:** edge Lua on HTTP/2 listener or tracker H2 parser before flattening to `parsedHTTPRequest`.

**Target:**

- Encode order as compact token (e.g. `mahp` vs `mphas`) forwarded on `X-H2-Pseudo-Order` or stored on connection context for first request.
- Mismatch vs UA-family expected order → L2-weak `h2_pseudo_order_mismatch`.

### Done gates

- [x] Holdout per browser family row (`TestH2PseudoOrder_holdoutChromeVsFirefox`)
- [x] HTTP/1.1 and HTTP/3 paths unchanged (no false positive; curl UA skipped)
- [x] Document edge-only requirement when tracker sees re-encoded H2 from nginx proxy (`ANTIFRAUD.md` CDN limits)

---

## `h2_h1_downgrade_artifact`

**Priority:** P2

**Gap:** Proxy tunnel H2→H1.1→H2 can leak `connection`, `keep-alive`, `transfer-encoding` or wrong header casing into H2-decoded requests.

**Surface:** `internal/ingestion/http2_hpack.go` / `parseH2Ingress` after decode.

**Target:**

- Reject or flag requests with forbidden H2 header names or mixed case on hot path (parser may already reject some; extend with explicit fraud signal `h2_downgrade_artifact`).
- Optional: nginx `edge-phase2.lua` parity check.

### Done gates

- [x] Dedicated holdout with synthetic downgrade corpus (`TestH2DowngradeArtifact_holdoutConnectionHeader`)
- [x] No regression on valid browser H2 corpus (`TestH2Ingress_captureChromeFingerprint`)

---

## `http1_header_order_fingerprint`

**Priority:** P1

**Gap:** `http1AssignHeader` is order-agnostic. Script clients (Python `requests`, Go `net/http`) permute canonical browser order.

**Surface:** `internal/ingestion/handler_http1_fsm.go` — append-only order index per known header token during DFA scan.

**Target:**

- Bounded order vector (max 32 slots, fixed header enum) on `parsedHTTPRequest`.
- Compare to UA-family canonical templates (Chrome desktop navigation GET/POST subsets).
- L2-weak `header_order_mismatch`; skip when `Sec-Fetch-*` absent and UA is non-browser (curl allowlist).

### Done gates

- [x] `http1IngressCanonical` corpus extended with order dimension; `TestChaos_CrossHop_NginxGnet` in `handler_http1_ingress_corpus_test.go`
- [x] Order capture covered by `http1_header_order_test.go` (`TestHTTP1HeaderOrder_parseHTTP1CapturesOrder`)

---

## `sec_fetch_metadata_validate` (shipped)

**Priority:** P1

**Gap:** Zero `Sec-Fetch-Site|Mode|Dest` handling. API `POST /track` with `Sec-Fetch-Mode: navigate` or missing group is common bot mistake.

**Surface:** `handler_http1_fsm.go` + H2 pseudo-header path; `domain.Event` optional fields; `DeviceFilter` or dedicated `SecFetchFilter`.

**Target:**

- Parse three headers on `/track` and `/click`.
- Rules (configurable per campaign preset `enhanced_defense`):
  - `POST /track`: expect `Sec-Fetch-Mode` in `cors|no-cors|same-origin`, `Sec-Fetch-Dest` `empty`; flag `navigate`+`document` on JSON POST.
  - Missing all three on modern Chrome UA (version parse from `Sec-CH-UA` or UA) → L2-weak `sec_fetch_missing`.
- Fail-open for non-browser UA markers and in-app WebView (`social_in_app` preset bypass).

### Done gates

- [x] Holdout: Chrome-like UA without Sec-Fetch → signal; WebView FBAN → no signal (`l7_wire_filter_test.go`)
- [x] OpenRTB `/openrtb/bid` excluded from Sec-Fetch rules (`L7WireFilter` path scope)
- [x] `ANTIFRAUD.md` + `fraudReasonRegistry` entry

---

## `client_hints_platform_matrix` (shipped)

**Priority:** P1

**Gap:** Only `deviceHintsMismatch` (`Sec-CH-UA` contains Chrome but UA does not). No `Sec-CH-UA-Platform`, `-Mobile`, `-Platform-Version`, greased brand parity, or high-entropy response after `Accept-CH`.

**Surface:** FSM parse + `DeviceFilter`; optional edge `Accept-CH` on safe page stub only (cold).

**Target:**

- Parse `Sec-CH-UA-Mobile`, `Sec-CH-UA-Platform` (structured header subset, no full RFC8941 parser on hot path if avoidable).
- Rules: Windows UA + `"Linux"` platform; mobile UA + `?0` mobile hint; Chrome >100 without any CH headers on HTTPS.
- Extend `device_mismatch` or add `client_hints_mismatch` with separate weight.

### Done gates

- [x] Table tests from real browser capture fixtures (`l7_wire_filter_test.go`, `TestClientHintsPlatform_holdoutWindowsUALinuxPlatform`)
- [x] Fail-open when CDN strips Client Hints (no `Sec-CH-UA` at all)

---

## `accept_encoding_browser_consistency`

**Priority:** P2

**Gap:** shipped — `Accept-Encoding` bitmask on `domain.Event`; `L7WireFilter` signal `accept_encoding_mismatch`.

**Surface:** `fillWireMetadataFromRequest` + `L7WireFilter`.

**Target:**

- Copy normalized encoding set to event (bitmask, no per-request string alloc).
- Modern Chrome UA without `br` or missing `zstd` (Chrome major >= 123) → L2-weak `accept_encoding_mismatch`.

### Done gates

- [x] Version-gated corpus with holdout for Chrome 128+ vs script client (`accept_encoding_match_test.go`)
- [x] OpenRTB path unchanged (`gzipAccepted` in `openrtb_exchange.go`)

---

## `accept_language_geo_hot_signal`

**Priority:** P2

**Gap:** shipped — `GeoFilter` signal `accept_lang_geo_mismatch` when global `ACCEPT_LANG_GEO_ENABLED` and campaign `accept_lang_geo_enabled`.

**Surface:** `GeoFilter` + `ensureIngestGeo`; campaign fraud PATCH.

**Target:**

- Light heuristic: primary lang tag vs GeoIP country (e.g. `pt-BR` IP in DE without `de` in list) → L2-weak `accept_lang_geo_mismatch`.
- Campaign knob `accept_lang_geo_enabled`; default off; `enhanced_defense` preset enables.

### Done gates

- [x] Holdout: matching lang+geo passes; obvious mismatch fails (`accept_lang_geo_match_test.go`)
- [x] CGNAT / mobile carrier policy does not bypass this signal (not in `shouldBypassCGNATIPVelocity` scope)

---

## `tls_alpn_browser_mismatch` (shipped)

**Priority:** P1

**Gap:** ALPN not extracted in `edge-tls-fingerprint.lua`; cannot flag Chrome UA with ALPN `http/1.1` only (no `h2`).

**Surface:** `deploy/nginx/lua/edge-tls-fingerprint.lua` → `X-TLS-ALPN`; tracker `DeviceFilter`.

**Target:**

- Forward ALPN list from ClientHello (comma-separated or first protocol).
- Chrome 100+ with only `http/1.1` → L2-weak `tls_alpn_mismatch` combined with UA check.

### Done gates

- [x] ALPN extraction in `deploy/nginx/lua/edge-tls-fingerprint.lua` → `X-TLS-ALPN` via `edge-ingress.lua`
- [x] Fail-open without `X-TLS-ALPN` (CDN path; `L7WireFilter` skips when header absent)

---

## `tls_ja4_browser_corpus`

**Priority:** P1

**Gap:** shipped — JA4 a-section prefix corpus with UA browser family matrix; signal `tls_ja4_mismatch`.

**Surface:** `tls_ja4_browser_corpus.go` + `DeviceFilter`; edge computes JA4 (`edge-tls-fingerprint.lua`).

**Target:**

- Versioned snapshot: JA4 prefix → allowed UA families (`chrome`, `firefox`, `safari`, `okhttp`, `go`).
- Known prefix + UA family not in allow mask → L2-weak `tls_ja4_mismatch`.
- Bundled corpus `deploy/vendor/fixtures/tls_ja4_browser_corpus.txt`; optional overlay `ja4_browser_corpus.txt` in `TLS_FINGERPRINT_FEED_DIR`.

### Done gates

- [x] Holdout: corpus Chrome JA4 + Chrome UA pass; Go TLS JA4 prefix + Chrome UA fail
- [x] `FuzzJA3Parse` extended for JA4 corpus lines
- [x] ECH / CDN limits in `ANTIFRAUD.md`

---

## `tcp_mss_asn_tunnel_heuristic`

**Priority:** P2

**Gap:** shipped — `TCPMSSFilter` tunnel heuristic `tcp_tunnel_mss` when MSS below threshold on residential ASN (not mobile carrier or DC ASN table).

**Surface:** `tcp_mss_filter.go` + GeoIP ASN snapshot on hot path.

**Target:**

- When `TCPMSS < TCP_MSS_TUNNEL_THRESHOLD` (default 1400) and ASN is generic residential ISP (not mobile carrier table, not DC ASN feed) → `tcp_tunnel_mss`.
- Edge high-byte MSS (`X-TCP-MSS` 0–255) decoded as `value << 8`; full MSS values accepted when edge forwards them.

### Done gates

- [x] Holdout: home fiber ASN + normal MSS passes; same ASN + MSS 1280 flags (`tcp_mss_filter_test.go`)
- [x] Fail-open without `X-TCP-MSS`

---

## `tcp_p0f_signature_correlation`

**Priority:** P2

**Gap:** shipped — XDP `hash_tcp_syn_fields` forwarded as `X-TCP-SIG`; tracker corpus maps hash to allowed UA families; signal `tcp_syn_os_mismatch`.

**Surface:** `edge_filter.c` observe-only fingerprint ringbuf; `edge-tcp-fp-sync.lua` + `edge-ingress.lua`; `tcp_syn_sig_corpus.go`.

**Target:**

- Compact SYN signature at XDP (`ttl`, `window`, `mss`, `doff`); Redis `tcp_hash`; edge `X-TCP-SIG`.
- Mismatch OS class vs UA family → `tcp_syn_os_mismatch` (L2-weak).

### Done gates

- [x] `TestXDP_fingerprintDoesNotCauseDrop` unchanged (observe only)
- [x] Redis staging schema in `edge.mdc`

---

## `track_json_serialization_fingerprint`

**Priority:** P2

**Gap:** shipped — byte-scan on native JSON `/track` detects sorted top-level keys, Python `": "` spacing, and 16+ digit `ts`/`timestamp` integers; signal `json_serialization_bot`.

**Surface:** `track_json_serialization_scan.go`; flags on `domain.Event`; `JSONSerializationFilter`.

**Target:**

- Sorted top-level keys (Go `encoding/json` signature).
- Python-style `": "` spacing when body length <= 4096.
- Timestamp field 16+ digit integer → `json_serialization_bot`.
- Campaign `json_serialization_enabled`; global `JSON_SERIALIZATION_FINGERPRINT_ENABLED` default off.

### Done gates

- [x] Holdout: browser-order fixture passes; sorted-key fixture fails (`track_json_serialization_scan_test.go`)
- [x] Protobuf and ORTB3 ingress unaffected (scan only on native JSON path)
- [x] No `json.Unmarshal` on hot path (byte scan only)

---

## `track_behavior_telemetry_ingest`

**Priority:** P2

**Gap:** Mouse bezier and event counts only on `POST /track/verify`; `/track` has no keystroke / pointer payload.

**Surface:** Optional `telemetry` object on track JSON schema; `safe_page_attestation_advanced.go` logic reused.

**Target:**

- Optional array `telemetry.events[]` with `t`, `ts`, `x`, `y` (cap 64 events, parse budget).
- Hot path: zero events on conversion POST with human UA → L2-weak `behavior_telemetry_missing` when campaign `enhanced_defense`.
- Reuse `checkBezierBot` for submitted mouse streams.

### Done gates

- [x] OpenAPI + parser budget tests
- [x] Holdout: empty telemetry flags when required; real curve passes
- [x] Static client snippet `internal/ingestion/track_telemetry.js` (`trackTelemetryArm`, `trackTelemetrySnapshot`)

---

## `rtt_split_tunnel_detect`

**Priority:** P3

**Gap:** No measurement of delta between TCP SYN-ACK RTT and time-to-first-application-byte; residential tunnels add multi-hop latency.

**Surface:** edge nginx (`$connection_time` vs request time) or XDP timestamp ringbuf; cold aggregation acceptable.

**Target:**

- Per-connection: `rtt_syn_ms`, `ttfb_app_ms`; flag when `ttfb_app_ms - rtt_syn_ms` exceeds threshold with low jitter variance (configurable).
- Start as **cold** CH feature for `ivt-detector`; hot signal only after false-positive study.

### Done gates

- [x] CH column or `ml_features_1m` extension documented
- [x] No sync CH read on `/track`

---

## `residential_intel_hot_path_read`

**Priority:** P1

**Gap:** `residential_intel_enricher` writes ClickHouse and feed file; hot path `residential_proxy` uses ring heuristic only, not `intel:residential:` cache.

**Surface:** `internal/ingestion/residential_proxy_ring.go` + `SettingsWatcher` snapshot from Redis.

**Target:**

- Periodic snapshot load of residential intel bitset or LPM (same pattern as proxy VPN feed).
- On match → `residential_proxy` signal without waiting for ring velocity.
- SKU gate `external_residential_intel` (`sku.yaml`).

### Done gates

- [x] Integration test with testcontainers Redis + enricher fixture row
- [x] Fail-open when cache empty or stale (TTL metric)
- [x] `ANTIFRAUD.md` row updated: hot read path named

---

## `cross_layer_fraud_score_snapshot`

**Priority:** P2

**Gap:** L4/TLS/L7 signals accumulated as independent weights in `filter_context.go`; no explicit "desync count" for operator dashboards.

**Surface:** `filter_context.go`, fraud stream payload, optional admin report.

**Target:**

- Derived field `layer_desync_count` on fraud reject stream: count of mismatched layers among {tcp_os, tls_ja4, client_hints, sec_fetch, h2} that fired on same event.
- Admin report `GET /api/v1/reports/layer-desync-summary` (cold, CH aggregate): **shipped** (`reports_layer_desync_summary.go`, CH `00028_fraud_events_layer_desync.sql`).

### Done gates

- [x] Holdout: single-layer signal → count 1; multi → count N (`layer_desync_test.go`)
- [x] Grafana panel doc in `ANTIFRAUD.md` (fraud-stream consumer; admin report noted as pending)

---

## Suggested implementation order (completed 2026-08-27)

All tiers shipped. Historical order:

1. Edge-forward prerequisites: `tls_alpn_browser_mismatch`, `h2_settings_frame_fingerprint`, `h2_pseudo_header_order`.
2. Hot L7 quick wins: `sec_fetch_metadata_validate`, `client_hints_platform_matrix`, `http1_header_order_fingerprint`.
3. Strengthen existing: `tls_ja4_browser_corpus`, `residential_intel_hot_path_read`.
4. Partner-sensitive / opt-in: `track_json_serialization_fingerprint`, `track_behavior_telemetry_ingest`.
5. Research tier: `rtt_split_tunnel_detect` (cold only).

---

## Explicit non-goals

| Item | Stance |
| :--- | :--- |
| Per-IP residential proxy guarantee | Rotating proxies evade L3/L4; document honestly (`ANTIFRAUD.md`) |
| ML on hot path for wire features | Batch `fraud-scorer` / `ivt-detector` only |
| Full structured-headers RFC8941 parser on `/track` | Bounded scans only (`hot-path.mdc`) |
| Block solely on missing Sec-Fetch for all traffic | Default off; breaks non-browser postbacks |

---

## Verification (backlog-wide)

Executed 2026-08-27 (doc close pass):

```bash
go build -o /dev/null ./cmd/tracker/                                    # exit 0
go test ./internal/ingestion/ -short \
  -run 'Device|TCPMSS|OSFingerprint|SecFetch|H2|JSON|LayerDesync|AcceptEncoding|AcceptLang|HeaderOrder|BehaviorTelemetry|Residential' \
  -count=1                                                                # exit 0, ok 0.293s
go test ./internal/edge/... -short -count=1                               # exit 0, ok 2.008s
bash scripts/ci/antifraud_doc_gate.sh                                     # exit 0
bash scripts/ci/check_no_legacy_naming.sh                                 # exit 0
```

Not run in this pass: `make test-alloc-gate` (no hot-path code changed). Run after future filter edits.
