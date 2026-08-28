# Antifraud reference

Operator reference for tracker fraud layers, signals, and cold-path ML. Implementation: `internal/ingestion`, `internal/fraud`, `internal/edge`, `cmd/ivt-detector`, `cmd/fraud-scorer`. Architecture: [ARCHITECTURE.md](../../docs/ARCHITECTURE.md). Edge detail: `.cursor/rules/edge.mdc`.

---

## Hot path vs cold path

| Path | Scope | Sync I/O |
| :--- | :--- | :--- |
| Hot | `/track`, `/click`, `/tg/*` accept/reject | Redis: at most one `EVALSHA` per accepted event (zero when local quanta full-skip eligible). No Postgres, ClickHouse, or ML inference on request thread. |
| Cold | Admin `:8188`, processor `:8186`, `ivt-detector`, `fraud-scorer` | Postgres settlement, ClickHouse analytics, outbox to Redis, batch ML. |

`/openrtb/bid` does not run the full `FilterEngine` chain. Fraud filters on that path are out of scope here.

Handler SLA (load test / CI): `ad_http_request_duration_seconds` p95 < 50 ms, p99 < 80 ms (hard ceiling 100 ms). Redis unified-filter Lua: p99 < 10 ms per shard. These are end-to-end budgets, not single-function timings.

ML (LightGBM/ONNX) runs only in `cmd/fraud-scorer`. Tracker reads pre-published boosts from a Go snapshot (`SettingsWatcher.GetFraudScoreBoosts`, Redis key prefix `ml:score:boost:`). SKU flag: `ml_fraud_boost` (`deploy/vendor/sku.yaml`).

`BenchmarkFilterFraudBoost` measures isolated `FilterEngine.Check` with a warmed in-memory boost map (~90 ns/op in CI). It does not include Redis Lua, Postgres, or full filter chain latency.

---

## Filter chain and layer decision

`FilterEngine.checkInner` (`filters.go`) runs local filters in order. When the next filter is `UnifiedFilter` and the event already has fraud signals (`acc.count > 0`), `applyFraudLayerDecision` runs first:

| Layer outcome | `UnifiedFilter` / Lua | Main event stream | Budget debit |
| :--- | :--- | :--- | :--- |
| None | Runs | Yes (if other filters pass) | Yes (if accepted) |
| L2 shadow | Skipped (`break` before `UnifiedFilter`) | Yes | No |
| L1 reject | Skipped (`ErrFraudDetected`) | No | No |

Layer logic: `decideFraudLayer` / `applyFraudLayerDecision` in `filter_layer.go`.

Fraud reject telemetry uses a separate Redis stream (`FraudStreamWriter`, default name `fraud-stream`). Both silent and hard L1 rejects call `enqueueFraudReject` on `/track` and `/click` (`handler.go`, `track_ingest_gnet.go`, `click_redirect.go`).

---

## Reaction levels (L1 / L2 / L3)

| Level | Trigger (code) | HTTP | Streams / analytics |
| :--- | :--- | :--- | :--- |
| L1 reject | L3 signal, or >= 2 L1-high signals | See `silent_reject_enabled` below | Fraud reject stream only; no main postback/event stream; no budget debit |
| L2 shadow | One L1-high, any L2-weak, or tier suspect/ivt/block | Normal accept (202 `/track`, 302 `/click`) | Main stream with `ShadowEvent=true`; ClickHouse via processor; no unified Lua debit |
| L3 blacklist | IP on Redis set `blacklist:fraud` | Contributes to L1 when combined with other signals | Signal `l3_blocklist` (weight 100). Edge may 403 from nginx blacklist cache before tracker |

Score tiers (from accumulated signal weights + optional ML boost): `MapFraudTier` in `filter_context.go`. Campaign thresholds `fraud_threshold_pass|suspect|ivt|block`; defaults 30 / 60 / 80 / 100 (`internal/domain/campaign.go`).

### `silent_reject_enabled`

Campaign flag in Postgres and registry snapshot. Hot path reads `camp.SilentRejectEnabled` in `fraudTrackOutcome` (`track_core.go`).

| Flag | `/track` | `/click` |
| :--- | :--- | :--- |
| `true` | HTTP 202; `evt.SilentRejectEvent=true`; fraud reject stream | 302 to safe page URL, or DMR HTML stub (`/safe_page_stub`); never bare 204 |
| `false` | HTTP 403 (`filterRejectFraudBlocked`); fraud reject stream | HTTP 403; fraud reject stream |

Legacy JSON field `ghost_ivt_enabled` is accepted on PATCH for one release; canonical field is `silent_reject_enabled`.

ClickHouse column: `silent_reject_event` on `fraud_events` (renamed from `ghost_event` in migration `00022_silent_reject_event_column.sql`). Report: `GET /api/v1/reports/silent-reject-impression-funnel` (`reports_silent_reject_impression.go`). Legacy URL alias: `/api/v1/reports/ghost-impression-funnel`.

---

## Fraud signals (hot path)

Registry: `fraudReasonRegistry` in `filter_errors.go`.

| Code | Weight | Layer flag | Notes |
| :--- | :---: | :--- | :--- |
| `datacenter_ip` | 45 | L1-high | GeoIP / DC ASN tables |
| `low_ttc` | 45 | L1-high | `applyGoTTC` in Go before Lua |
| `tls_blocklist` | 45 | L1-high | JA3/JA4 tables; social in-app WebView bypass when `social_in_app_enabled` |
| `l3_blocklist` | 100 | L3 | `FraudBlacklistFilter`; 128-shard in-memory TTL cache (5 s, max 2048 entries/shard); `SISMEMBER` on miss; Redis error fail-open |
| `missing_imp_ts` | 35 | L2-weak | |
| `device_mismatch` | 35 | L2-weak | UA vs `Sec-CH-UA` |
| `sec_fetch_anomaly` | 35 | L2-weak | `Sec-Fetch-*` missing or navigate+document on JSON POST |
| `client_hints_mismatch` | 35 | L2-weak | `Sec-CH-UA-Platform` / `-Mobile` vs UA family |
| `tls_alpn_mismatch` | 35 | L2-weak | Chrome UA with ALPN `http/1.1` only (no `h2`) |
| `h2_settings_mismatch` | 35 | L2-weak | Client SETTINGS / WINDOW_UPDATE vs Chrome UA |
| `h2_pseudo_order_mismatch` | 35 | L2-weak | HPACK pseudo-header order vs Chrome UA |
| `h2_downgrade_artifact` | 35 | L2-weak | Forbidden H1 headers or mixed case on HTTP/2 wire |
| `header_order_mismatch` | 35 | L2-weak | HTTP/1.1 header order vs Chrome template (requires `Sec-Fetch-*`) |
| `accept_encoding_mismatch` | 35 | L2-weak | Chrome UA `Accept-Encoding` missing `br` or `zstd` (version-gated) |
| `accept_lang_geo_mismatch` | 35 | L2-weak | `Accept-Language` vs GeoIP country (campaign `accept_lang_geo_enabled`) |
| `tls_ja4_mismatch` | 35 | L2-weak | JA4 a-section prefix vs UA browser family corpus |
| `tcp_mss_anomaly` | 35 | L2-weak | `evt.TCPMSS` / `X-TCP-MSS` high-byte below `TCP_MSS_ANOMALY_MIN_BYTE` |
| `tcp_tunnel_mss` | 35 | L2-weak | MSS below `TCP_MSS_TUNNEL_THRESHOLD` on residential ASN (not mobile/DC) |
| `tcp_syn_os_mismatch` | 35 | L2-weak | XDP SYN hash (`X-TCP-SIG`) vs UA family corpus |
| `json_serialization_bot` | 35 | L2-weak | Native JSON `/track` allocator fingerprint (campaign opt-in) |
| `behavior_telemetry_missing` | 35 | L2-weak | Conversion POST without `telemetry.events` when `safe_page_enabled` and `attestation_enabled` (`BEHAVIOR_TELEMETRY_ENABLED`) |
| `behavior_bezier_bot` | 35 | L2-weak | Linear uniform mousemove stream in `telemetry.events` (`checkBezierBot`) |
| `os_fingerprint_mismatch` | 35 | L2-weak | TTL vs UA family; disable behind CDN (`edge.mdc`) |
| `ipv4_rotation` | 35 | L2-weak | Subnet velocity tables |
| `residential_proxy` | 35 | L2-weak | Ring velocity heuristic (`RESIDENTIAL_PROXY_HOT_ENABLED`) or intel LPM snapshot (`RESIDENTIAL_INTEL_HOT_READ_ENABLED`, SKU `external_residential_intel`, `intel:residential:` + `external_residential.txt`) |
| `moderator_ip` | 45 | L1-high | Signed `moderator_intel_v1` snapshot; per-campaign `moderator_intel_enabled` (default off) |
| `attestation_missing` | 35 | L2-weak | Light attestation on `/click` |

Safe page `/track/verify` only (not `/click` redirect): canvas test-retest when `canvas_retest_enabled` on campaign (`PATCH /api/v1/campaigns/{id}/fraud`). Client sends `canvas_hash_a` and `canvas_hash_b`; mismatch code `canvas_retest_mismatch` returns safe view. Fail-open when only one hash present. Default flag off (Firefox/WebKit privacy noise).

Admin presets: `conservative`, `balanced`, `aggressive`, `enhanced_defense`, `social_in_app` (`service_fraud_presets.go`). Campaign fraud PATCH: `/api/v1/campaigns/{id}/fraud`.

### CDN and TLS fingerprint limits

TCP MSS, TTL, and JA3/JA4 at the tracker reflect edge-terminated connections when traffic passes Cloudflare, ALB, or other L7 terminators. Edge forwards `X-TCP-MSS`, `X-TCP-TTL`, `X-TLS-JA3`, `X-TLS-JA4` when `edge-tcp-fp-sync` / XDP fingerprint path is active.

| Signal | Direct edge (nginx terminates TLS, TCP FP sync) | CDN / ALB without edge TCP FP |
| :--- | :--- | :--- |
| `tcp_mss_anomaly` | Active when `TCP_MSS_ANOMALY_ENABLED` and `X-TCP-MSS` set | Fail-open: no header, no signal (`tcp_mss_filter.go`) |
| `tcp_tunnel_mss` | Active when `TCP_MSS_TUNNEL_ENABLED`, `X-TCP-MSS` set, and GeoIP ASN resolves | Fail-open: no header or ASN lookup miss |
| `tcp_syn_os_mismatch` | Active when `TCP_SYN_SIG_ENABLED` and `X-TCP-SIG` set | Fail-open: no header or unknown hash in corpus |
| `rtt_split_tunnel` (cold) | `X-RTT-SYN-MS` + `X-TTFB-APP-MS` on edge; columns `clicks.rtt_split_delta_ms` | Fail-open: no headers; `ivt-detector` rule `rtt_split_tunnel` only (no hot `/track` signal) |
| `os_fingerprint_mismatch` | Active when `OS_FINGERPRINT_MISMATCH_ENABLED` and `X-TCP-TTL` set | Fail-open: `ad_os_fingerprint_skipped_total{reason="no_tcp_headers"}`; or set `OS_FINGERPRINT_MISMATCH_ENABLED=false` |
| TLS JA3/JA4 L1 block | Edge TLS fingerprint on `/click` | CDN JA3 only; use allowlists, not standalone block |
| `device_mismatch` (UA vs TLS) | Meaningful when edge forwards client TLS hints | Not reliable as client JA3 proof behind CDN |
| `tls_alpn_mismatch` | Active when `TLS_ALPN_MISMATCH_ENABLED` and `X-TLS-ALPN` set | Fail-open without header |
| `sec_fetch_anomaly` / `client_hints_mismatch` | Browser headers on direct edge | CDN may strip Client Hints or Sec-Fetch; fail-open when absent |

JA3/JA4 spoofing and ECH reduce standalone TLS block efficacy. JA4 browser corpus (`tls_ja4_mismatch`) matches only the JA4 a-section prefix (before first `_`) against bundled `deploy/vendor/fixtures/tls_ja4_browser_corpus.txt`; unknown prefixes fail-open. CDN-terminated TLS forwards edge JA4 via `X-TLS-JA4`; corpus is ineffective when header absent.

---

## Presets

### `social_in_app`

- `conn_type_policy` may restrict to mobile carriers when configured.
- TLS block on `/click` skipped for in-app WebView UA markers (`FBAN`, `FBAV`, `musical_ly`, `Instagram`) when `social_in_app_enabled` (`landing_tls_fingerprint_hook.go`).
- Attestation and other filters still run.

### `enhanced_defense`

`applyEnhancedDefensePreset` (`service_fraud_enhanced_defense.go`) sets: safe page, `silent_reject_enabled`, click redirect delivery, attestation (strict), network blocks (proxy/VPN, TLS fingerprint, L1 CIDR), signed links.

L1 fraud on `/click`: 302 to safe page when configured; DMR HTML stub otherwise. In-place 200 only for attestation probe (`forceSafe`).

---

## Go filters before Redis Lua

Placement and fraud blacklist run inside `UnifiedFilter.Check` before `EVALSHA`. `EntitlementsFilter` runs in `FilterEngine` before `UnifiedFilter` (`cmd/tracker/main.go`). Lua `unified-filter.lua` handles atomic budget, dedup, pacing, and consolidated gates. Ingress RPD, placement `HEXISTS`, and fraud `SISMEMBER` are not in the script body (`lua_precheck_test.go`).

| Check | Implementation | Redis on steady path |
| :--- | :--- | :--- |
| Placement blacklist | `PlacementBlacklistFilter` in `UnifiedFilter.Check` | 128-shard in-memory TTL cache (5 s, max 2048 entries/shard); `HEXISTS` on miss |
| Fraud blacklist | `FraudBlacklistFilter` in `filter_layer.go` | 128-shard in-memory TTL cache (5 s, max 2048 entries/shard); `SISMEMBER` on miss |
| Ingress RPD | `EntitlementsFilter` + `SetIngressRPDHandledExternally(true)` in `cmd/tracker/main.go` | One `INCR` + `EXPIRE` per event on `ingress:day:{region?}{customer_id}:{date}`. Flag skips `UnifiedFilter.checkIngressRPDGo` so Lua precheck does not double-count |
| TTC signal | `applyGoTTC` in `UnifiedFilter.Check` | No Lua TTC branch when computed in Go |
| ML boost | `GetFraudScoreBoosts()` in fraud filter | None on read; `ml:score:boost:*` synced async via `SettingsWatcher` (pub/sub + `FRAUD_BOOST_FULL_RESYNC_SEC`, default 10s). Processor `MicroBatcher` flush default 50ms (`FRAUD_MICROBATCH_FLUSH_MS`).

Local quanta (`LOCAL_QUOTA_MODE=live`) can eliminate sync `EVALSHA` for eligible traffic.

Fraud stream aggregation (`fraud_stream_queue.go`): L3 and dual-L1 events are never aggregated (`fraudAggregateExempt`); weak L2 signals may batch into `fraud_aggregate` events.

Cross-layer desync (`layer_desync_count` on fraud-stream `AdStreamEvent`): count of distinct wire layers among `{tcp_os, tls_ja4, client_hints, sec_fetch, h2}` that fired on the same reject. Derived in `layer_desync.go` from `fraudAccumulator` signals at `applyFraudAccumulatorForCampaign`. Multiple H2 reasons (`h2_settings_mismatch`, `h2_pseudo_order`, `h2_downgrade_artifact`) count as one `h2` layer. Non-wire signals (datacenter IP, TTC, blacklist) do not increment the count.

**Admin report:** `GET /api/v1/reports/layer-desync-summary` aggregates `fraud_events.layer_desync_count` by campaign (`reports_layer_desync_summary.go`). Requires CH migration `00028_fraud_events_layer_desync.sql`.

**Grafana panel (fraud-stream consumer):** after the processor unmarshals `AdStreamEvent`, expose `layer_desync_count` as a Prometheus histogram or heatmap bucketed 0–5. Example PromQL on a custom counter labeled by `layer_desync_count` (emit from fraud-stream consumer metric hook): `sum by (layer_desync_count) (rate(ad_fraud_stream_layer_desync_total[5m]))`. Alert when share with `layer_desync_count >= 3` exceeds baseline (multi-layer bot / proxy toolchain).

### Fraud evidence pack (CPA dispute export)

`GET /api/v1/reports/fraud-evidence-pack?click_id={id}` returns a signed JSON bundle for network CPA disputes: CH timeline (impression/click/conversion), `fraud_events` rows (reason, score, `layer_desync_count`, silent reject), and deduplicated signal summary. HMAC-SHA256 over canonical JSON (`digest_sha256` + `signature` fields).

| Env | Default | Role |
| :--- | :--- | :--- |
| `FRAUD_EVIDENCE_PACK_HMAC_SECRET` | falls back to `CONSENT_HMAC_SECRET` | Signing key for dispute bundle |
| Permissions | `audit:read` + `campaigns:read` | Same as fraud-breakdown |

No raw IP/UA in bundle (PII hashed at ingest). Empty timeline and fraud rows → `404 NOT_FOUND`. Signing secret unset → `503 EVIDENCE_SIGNING_UNAVAILABLE`.

Buyer-facing dispute surfaces: `customer_fraud_dispute_evidence` (cold path API); operator signed export remains `fraud-evidence-pack` above. Cross-check `layer_desync_count` with `GET /api/v1/reports/layer-desync-drilldown` aggregates.

---

## Edge and XDP (optional)

License feature `ebpf_xdp_edge` (Enterprise SKU only).

| Component | Role |
| :--- | :--- |
| Nginx `access-check.lua` | Rate limit, circuit breaker, `ngx.shared.blacklist_cache`, proxy to tracker pool |
| `edge-blacklist-sync.lua` | Redis shard 0 -> shared dict; changelog (cap `EDGE_BLACKLIST_CHANGELOG_MAX_IPS`, default 64) + periodic full sync (`EDGE_BLACKLIST_SYNC_INTERVAL_SEC`) |
| `edge-xdp` | XDP attach; L3/L4 drop for listed hosts/CIDR |
| `edge-bpf-sync` | Redis -> BPF maps (LRU host hash, LPM trie; `max_entries` per map spec in `edge.mdc`) |

XDP drops known bad IPs and syn-flood patterns at the NIC. It does not classify residential proxy rotation or application-layer fraud. LRU eviction under map pressure: `TestFault_XDPLRUEvictionUnderPressure` in `internal/edge`.

**XDP ops metrics** (`edge-bpf-sync` `:9090/metrics`):

| Metric | Meaning |
| :--- | :--- |
| `edge_blocklist_map_fill_ratio{map}` | BPF map occupancy / `max_entries`; alert above 0.9 on `blocklist_host_v4` |
| `edge_blocklist_lru_eviction_total{map}` | Host inserts while LRU map full |
| `edge_blocklist_changelog_lag_seconds` | Seconds since last consumed changelog score; rising lag = sync falling behind |

Rollback: stop `edge-bpf-sync` and `edge-xdp`; nginx Lua blacklist remains.

Auto-blocks enter BPF maps via control outbox -> Redis shard 0 -> sync only (`compliance.mdc`). Workers do not open outbound connections to visitor IPs.

---

## Cold path workers

Compose profile `analytics-ml` adds `ivt-detector` and `fraud-scorer`. SKU flags: `ivt_ml_detector`, `ml_fraud_boost`.

### `cmd/ivt-detector`

Poll interval ~5 minutes on ClickHouse `ml_features_1m`. Rules: CTR spikes, fingerprint clusters, interval bots, `rtt_split_tunnel` (reads `clicks.rtt_split_delta_ms`; no hot-path signal). Pauses when outbox `PENDING` > 500.

| Env | Default | Role |
| :--- | :--- | :--- |
| `IVT_RTT_SPLIT_TUNNEL_ENABLED` | `true` | Cold `rtt_split_tunnel` rule in `ivt-detector` |
| `IVT_RTT_SPLIT_MIN_DELTA_MS` | `150` | Minimum `ttfb_app_ms - rtt_syn_ms` average per IP |
| `IVT_RTT_SPLIT_MAX_VARIANCE` | `2500` | Maximum `varPop(rtt_split_delta_ms)` per IP (low jitter tunnel) |
| `IVT_RTT_SPLIT_MIN_SAMPLES` | `5` | Minimum click rows with timing per IP in window |

Edge forwards `X-RTT-SYN-MS` (`$tcpinfo_rtt`) and `X-TTFB-APP-MS` (`$connection_time`). Tracker persists async to `clicks` / `conversions` / `impressions` (`00026_rtt_split_tunnel.sql`); no sync CH read on `/track`. Admin report: `GET /api/v1/reports/rtt-split-tunnel`. CDN-terminated TCP does not expose RTT headers to origin; treat `signals_degraded` / stale freshness as expected without edge timing (`edge.mdc`).

### Mobile biometrics (cold path)

Client snippet `internal/ingestion/track_biometrics.js`: arms `touchstart`/`touchmove`/`deviceorientation`, exposes `trackBiometricsSnapshot()` for `telemetry.events` on conversion `POST /track`. Hot path (`MOBILE_BIOMETRICS_ENABLED`, default `false`) summarizes touch count, gyro sample count, variance, and flat-orientation flag into `conversions` columns (`00027_mobile_biometrics.sql`). No ML on `/track`.

| Env | Default | Role |
| :--- | :--- | :--- |
| `MOBILE_BIOMETRICS_ENABLED` | `false` | Ingest summary on `/track` when `telemetry` present |
| `IVT_MOBILE_BIOMETRICS_ENABLED` | `true` | Cold `mobile_biometrics` rule in `ivt-detector` |
| `IVT_MOBILE_BIOMETRICS_MIN_SAMPLES` | `5` | Minimum mobile conversion rows per IP in window |
| `IVT_MOBILE_BIOMETRICS_MIN_FLAT_HITS` | `4` | Flat gyro emulator threshold (orientation range <= 2 deg) |
| `IVT_MOBILE_BIOMETRICS_MIN_MOTIONLESS` | `5` | Mobile rows with zero touch and zero gyro samples |
| `IVT_MOBILE_BIOMETRICS_MIN_GYRO_SAMPLES` | `3` | Minimum gyro samples for flat-orientation match |

ML enforcement actions (`EnqueueFraudThreatBatch`):

| Action | Outbox | Effect |
| :--- | :--- | :--- |
| `boost` | `ML_SCORE_BOOST` | Redis `ml:score:boost:{campaign_id}` |
| `blacklist` | `ML_BLACKLIST_ADD` | IP on `blacklist:fraud` (L3 signal on hot path) |
| `silent_reject` | `ML_SILENT_REJECT` | Per-IP `blacklist:fraud` add; legacy action alias `ghost` accepted on enqueue. Does **not** change `silent_reject_enabled` on campaigns |

### `cmd/fraud-scorer`

Batch scoring (up to 1000 events). Writes `ml:score:boost:{campaign_id}` on Redis; tracker subscribes via `SettingsWatcher`.

Admin ops: `/api/v1/ops/ml-model`, `/api/v1/fraud/labels`, `/api/v1/fraud/decisions`, `/api/v1/fraud/overrides`.

### Conversion smart reject (processor settlement)

Cold path only (`cmd/processor`). Runs before `ConversionPayoutApplier` and `ConversionPostbackEnqueuer.OnBatchStored`.

| Env | Default | Role |
| :--- | :--- | :--- |
| `CONVERSION_SMART_REJECT_ENABLED` | `true` | Master switch |
| `CONVERSION_REJECT_MIN_TTC_MS` | `3000` | Minimum click-to-conversion time |
| `CONVERSION_REJECT_NO_CLICK` | `true` | Reject empty `click_id` or missing CH click row |
| `CONVERSION_REJECT_LOW_TTC` | `true` | Reject faster than min TTC |
| `CONVERSION_REJECT_DUPLICATE` | `true` | Reject duplicate `campaign_id + click_id + goal_name` |
| `CONVERSION_REJECT_IP_DRIFT` | `true` | Reject when click country != conversion country |
| `CONVERSION_REJECT_DATACENTER_IP` | `false` | Reject datacenter IP at conversion (optional ASN checker) |
| `CONVERSION_REJECT_REPROCESS_ENABLED` | `true` | Replay deferred conversions after CH click lookup recovers |
| `CONVERSION_REJECT_REPROCESS_INTERVAL_MIN` | `15` | Reprocess tick interval (5-60 min) |
| `CONVERSION_REJECT_REPROCESS_LOOKBACK_HOURS` | `24` | CH window for pending conversions (1-168 h) |

Reason codes: `conversion_no_click`, `conversion_low_ttc`, `conversion_duplicate`, `conversion_ip_drift`, `conversion_datacenter_ip`.

On reject: set `FraudReason`; set `SilentRejectEvent` when campaign `silent_reject_enabled`; zero `revenue_micro` / `payout_micro` in payload. CH insert routes to `fraud_events` via `isFraudTelemetry`. Outbound CAPI/postback skipped when `FraudReason != ""`. PG settlement stats skip conversions with `FraudReason`.

Click lookup: one batched CH query per settlement batch (`clicks` + existing `conversions` goals). No per-event CH loop. When ClickHouse click store is unavailable (CH disabled or lookup error), CH-dependent rules are skipped; conversions with non-empty `click_id` are recorded with payload `conversion_validation_pending=true` and `revenue_micro` / `payout_micro` zero until reprocess validates (`ConversionPayoutApplier` skips pending rows). Empty `click_id` is still rejected when `CONVERSION_REJECT_NO_CLICK` is on. Datacenter IP rule still runs when configured (no CH click row required). PG settlement stats skip pending conversions. `ConversionPostbackEnqueuer` skips outbound postback while `conversion_validation_pending` is set (`ad_conversion_postback_deferred_total`).

Reprocess worker (`cmd/processor`): queries CH `conversions` with `conversion_validation_pending`, replays `ConversionRejectApplier`, deletes pending rows via `ALTER TABLE DELETE`, inserts rejects into `fraud_events`, inserts approved rows with payout resolved, then enqueues outbound postback. Per-campaign overrides: `conversion_reject_rules` on `PATCH /api/v1/campaigns/{id}/fraud` (unset fields inherit `CONVERSION_REJECT_*` env).

`CONVERSION_REJECT_DATACENTER_IP` requires processor GeoIP country DB plus optional ASN DB and DC ASN feed (`DC_ASN_HOT_ENABLED`); otherwise the rule is inactive.

---

## CGNAT mobile IP policy

When mobile carrier ASN is detected (GeoIP ASN on the request IP), skip **only** IP-frequency signals: `/click` `ipv4_rotation` safe-page redirect and ingress RPD `INCR`. Does not bypass `datacenter_ip`, TLS fingerprint blocklist, L3 blacklist, attestation, or UnifiedFilter Lua budget.

| Knob | Surface |
| :--- | :--- |
| `cgnat_ip_policy_enabled` | Campaign fraud PATCH / GET (`PATCH /api/v1/campaigns/{id}/fraud`) |
| `CGNAT_MOBILE_IP_BYPASS` | Tracker env global bypass (default `false`) |
| `CGNAT_MOBILE_CARRIER_ASNS` | Optional extra MNO ASNs appended to builtin tier-1 set |

Builtin carrier ASN snapshot: `internal/ingestion/mobile_carrier_asn_table.go` (excludes mobile proxy/reseller ASNs). Metric: `ad_cgnat_ip_bypass_total{signal="ipv4_rotation|ingress_rpd"}`.

---

## Verification

```bash
go test ./internal/ingestion/ -run 'Fraud|SafePage|TLS|SilentReject' -count=1
go test ./internal/edge/... -count=1
go test ./internal/controlplane/ -run 'Fraud|SilentReject|ML_BLACKLIST' -count=1
bash scripts/ci/compliance.sh
bash scripts/ci/antifraud_doc_gate.sh
```

Holdout: `TestFraudReject_holdoutSilentRejectFlag`, `TestFilterEngine_shadowSkipsUnifiedBudgetDebit_holdout`.

ROI slugs shipped 2026-08-26 (conversion reject, automation IVT metrics, canvas retest, CGNAT policy). L4/TLS/L7 desync: [residential_proxy_detection_backlog.md](./residential_proxy_detection_backlog.md) (**CLOSED**).
