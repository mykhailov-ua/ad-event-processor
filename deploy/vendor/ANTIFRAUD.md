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
| `tcp_mss_anomaly` | 35 | L2-weak | `evt.TCPMSS` / `X-TCP-MSS` from edge |
| `os_fingerprint_mismatch` | 35 | L2-weak | TTL vs UA family; disable behind CDN (`edge.mdc`) |
| `ipv4_rotation` | 35 | L2-weak | Subnet velocity tables |
| `residential_proxy` | 35 | L2-weak | Hot cache / intel enricher; SKU `external_residential_intel` |
| `moderator_ip` | 45 | L1-high | Signed `moderator_intel_v1` snapshot; per-campaign `moderator_intel_enabled` (default off) |
| `attestation_missing` | 35 | L2-weak | Light attestation on `/click` |

Admin presets: `conservative`, `balanced`, `aggressive`, `enhanced_defense`, `social_in_app` (`service_fraud_presets.go`). Campaign fraud PATCH: `/api/v1/campaigns/{id}/fraud`.

### CDN and TLS fingerprint limits

TCP MSS, TTL, and JA3/JA4 at the tracker reflect edge-terminated connections when traffic passes Cloudflare, ALB, or other L7 terminators. Edge forwards `X-TCP-MSS`, `X-TCP-TTL`, `X-TLS-JA3`, `X-TLS-JA4` when `edge-tcp-fp-sync` / XDP fingerprint path is active.

| Signal | Direct edge (nginx terminates TLS, TCP FP sync) | CDN / ALB without edge TCP FP |
| :--- | :--- | :--- |
| `tcp_mss_anomaly` | Active when `TCP_MSS_ANOMALY_ENABLED` and `X-TCP-MSS` set | Fail-open: no header, no signal (`tcp_mss_filter.go`) |
| `os_fingerprint_mismatch` | Active when `OS_FINGERPRINT_MISMATCH_ENABLED` and `X-TCP-TTL` set | Fail-open: `ad_os_fingerprint_skipped_total{reason="no_tcp_headers"}`; or set `OS_FINGERPRINT_MISMATCH_ENABLED=false` |
| TLS JA3/JA4 L1 block | Edge TLS fingerprint on `/click` | CDN JA3 only; use allowlists, not standalone block |
| `device_mismatch` (UA vs TLS) | Meaningful when edge forwards client TLS hints | Not reliable as client JA3 proof behind CDN |

JA3/JA4 spoofing and ECH reduce standalone TLS block efficacy.

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

Poll interval ~5 minutes on ClickHouse `ml_features_1m`. Rules: CTR spikes, fingerprint clusters, interval bots. Pauses when outbox `PENDING` > 500.

ML enforcement actions (`EnqueueFraudThreatBatch`):

| Action | Outbox | Effect |
| :--- | :--- | :--- |
| `boost` | `ML_SCORE_BOOST` | Redis `ml:score:boost:{campaign_id}` |
| `blacklist` | `ML_BLACKLIST_ADD` | IP on `blacklist:fraud` (L3 signal on hot path) |
| `silent_reject` | `ML_SILENT_REJECT` | Per-IP `blacklist:fraud` add; legacy action alias `ghost` accepted on enqueue. Does **not** change `silent_reject_enabled` on campaigns |

### `cmd/fraud-scorer`

Batch scoring (up to 1000 events). Writes `ml:score:boost:{campaign_id}` on Redis; tracker subscribes via `SettingsWatcher`.

Admin ops: `/api/v1/ops/ml-model`, `/api/v1/fraud/labels`, `/api/v1/fraud/decisions`, `/api/v1/fraud/overrides`.

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
