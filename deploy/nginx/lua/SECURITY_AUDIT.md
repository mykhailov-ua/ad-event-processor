# Edge Nginx Lua Security Audit

Date: 2026-09-06  
Scope: production modules under `deploy/nginx/lua/` (excluding `tests/` harnesses).  
Phases: access (`access_by_lua`), balance (`balancer_by_lua`), filter, worker timers.

Holdout tests: `edge_security_holdout_test.lua` (20+ cases, documents audit slugs below).  
DFA-specific holdouts: `edge_parse_dfa_fault_test.lua` (20 cases, regression holdouts).

Tests **pass while the documented gap is present** (holdout contract). After remediation, update or invert the corresponding case.

Run:

```bash
bash scripts/test/edge/lua_tests.sh security
bash scripts/test/edge/lua_tests.sh parse-dfa
```

## Methodology

1. Manual code review of all 27 production `.lua` modules.
2. Cross-check against Go parity contracts (`parser.mdc`, `http1TrackEdgePolicy`, `TestChaos_CrossHop_NginxGnet`).
3. Holdout tests that **fail if the vulnerability is accidentally "fixed" without an explicit contract change** (where applicable) or **pass while documenting exploitable behavior**.
4. Lightweight fuzz corpus (nested JSON, escape chains, varint bombs) asserting no worker crash in standalone luajit.

## Trust boundary model

| Input | Expected trust | Actual in code |
| :--- | :--- | :--- |
| `remote_addr` | Client IP (or `$realip` if configured upstream of edge) | Used for blacklist |
| `X-Client-ASN` | CDN/edge-injected only | Cleared on access; not used for bypass |
| Trusted ASN | CDN inner hop only | `EDGE_TRUSTED_ASN_HEADER` (default `X-Edge-Trusted-ASN`) |
| `X-Fraud-Score` | Tracker or ML sidecar | Cleared on access; not used for edge RL/block |
| Trusted fraud score | Internal hop only | `EDGE_TRUSTED_FRAUD_SCORE_HEADER` (default `X-Edge-Trusted-Fraud-Score`) |
| `X-Campaign-Id` | Internal hop only | Cleared on access; trusted header only with UUID normalize |
| POST body | Untrusted | DFA scan window 8192 bytes |

**Conclusion:** Client-spoofable headers are cleared or ignored on public edge; trusted inner-hop headers require CDN injection. All edge access findings remediated (2026-09-06).

---

## Severity legend

| Level | Meaning |
| :--- | :--- |
| Critical | Direct perimeter bypass or worker kill |
| High | Enforcement bypass with low attack cost |
| Medium | DoS, affinity manipulation, or conditional bypass |
| Low | Perf degradation, defense-in-depth gap |

---

## client_asn_blacklist_bypass: Spoofable `X-Client-ASN` skips IP blacklist — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `edge-asn.lua`, `access-check.lua`, `nginx.conf` |
| Phase | access |

### Fix (2026-09-06)

1. `edge_asn.sanitize_untrusted_asn_headers()` clears client `X-Client-ASN` before blacklist gate.
2. Bypass reads `EDGE_TRUSTED_ASN_HEADER` only (default `X-Edge-Trusted-ASN`).
3. `nginx.conf` exports `EDGE_TRUSTED_ASN_HEADER` env knob.

CDN: inject `X-Edge-Trusted-ASN` at the hop before nginx, or set `EDGE_TRUSTED_ASN_HEADER=X-Client-ASN` only when end clients cannot reach nginx directly.

### Holdouts

`tests/asn_trust_test.lua`, `client_asn_blacklist_bypass_client_asn_header_no_blacklist_bypass`.

---

## client_fraud_score_rl_tier: Spoofable `X-Fraud-Score` downgrades edge RL tier — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `edge-fraud-tier.lua`, `edge_track_policy.lua`, `nginx.conf` |
| Phase | access |

### Fix (2026-09-06)

1. `edge_fraud_tier.sanitize_untrusted_fraud_score_headers()` clears client `X-Fraud-Score`.
2. Edge RL and tier block read `EDGE_TRUSTED_FRAUD_SCORE_HEADER` only (default `X-Edge-Trusted-Fraud-Score`).
3. Missing trusted score defaults to 0 (pass tier); client header cannot force pass when trusted score says block.

### Holdouts

`tests/fraud_score_trust_test.lua`, `client_fraud_score_rl_tier_client_fraud_score_no_tier_bypass`.

---

## nil_campaign_id_ip_rl_fallback: Edge per-campaign RL skipped when `campaign_id` is nil — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge-rl.lua` |
| Phase | access |

### Fix (2026-09-06)

`edge_rl.rl_subject_key()` maps nil/empty `campaign_id` to `ip:{remote_addr}`. `allow()` always runs incr with the same tier/limit math; no early return on missing campaign id.

Truncated DFA / `{}` body without extractable id still hits per-IP edge RL bucket (alongside nginx `limit_req` on `/track`).

### Holdouts

`tests/edge_rl_fallback_test.lua`, `nil_campaign_id_ip_rl_fallback`.

---

## header_campaign_id_trusted_only: `X-Campaign-Id` controls routing without body proof — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge-campaign-id.lua`, `edge_track_policy.lua`, `nginx.conf` |
| Phase | access + balance |

### Fix (2026-09-06)

1. `edge_campaign_id.sanitize_untrusted_campaign_id_headers()` clears client `X-Campaign-Id` before routing/RL.
2. Stream/peek/openrtb fallbacks read `EDGE_TRUSTED_CAMPAIGN_ID_HEADER` only (default `X-Edge-Trusted-Campaign-Id`).
3. `resolve_campaign_id(body_id)` normalizes via `edge-uuid`; body DFA id wins when present.
4. `nginx.conf` exports `EDGE_TRUSTED_CAMPAIGN_ID_HEADER` env knob.

CDN inner hop: inject `X-Edge-Trusted-Campaign-Id` after body parse upstream, or set `EDGE_TRUSTED_CAMPAIGN_ID_HEADER=X-Campaign-Id` only when end clients cannot reach nginx directly.

### Holdouts

`tests/campaign_id_trust_test.lua`, `header_campaign_id_trusted_only_header_campaign_id_no_uuid_validation`.

---

## peek_socket_fail_policy: Peek/openrtb cosocket failure fail-open — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge_track_policy.lua` |
| Phase | access |

### Fix (2026-09-06)

Socket unavailable paths call `finish_policy_without_body()`: fraud tier check, `edge_campaign_id.resolve_campaign_id(nil)`, and `edge_rl.allow()` (IP fallback per nil_campaign_id_ip_rl_fallback). `apply_campaign_rl` always invokes `edge_rl.allow`, including nil campaign_id.

### Holdouts

`peek_socket_fail_policy_peek_fail_open_logic`.

---

## native_json_depth_limit: Native JSON DFA — unbounded `skip_json_value` recursion — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge-parse-dfa.lua` |
| Phase | access |

### Fix (2026-09-06)

`skip_json_value()` takes `(depth, max_depth)`. Native track JSON uses `MAX_NATIVE_JSON_DEPTH = 16` (Go `MaxJSONDepth` parity). OpenRTB walk keeps `MAX_JSON_DEPTH = 32`.

Depth > max returns `ERR_MALFORMED` before nested `{` / `[` recursion.

### Holdouts

`native_json_depth_bomb` in `edge_parse_dfa_fault_test.lua`, `native_json_depth_limit_native_json_depth_unbounded`, fuzz `fuzz_nested_native_json`.

---

## bl_pending_byte_cap: `_bl_pending` unbounded SHM growth — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge-blacklist-sync.lua`, `edge-metrics.lua` |
| Phase | worker timer (indirect edge impact) |

### Fix (2026-09-06)

1. `append_pending_ips()` caps `_bl_pending` at `PENDING_MAX_BYTES` (default 65536; env `EDGE_BLACKLIST_PENDING_MAX_BYTES`).
2. When cap exceeded and `_bl_ver > 0`, overflow IPs are **immediate-stamped** at current generation (bl_pending_overflow_immediate_stamp); not dropped.
3. `edge_metrics.record_blacklist_pending_immediate_stamp()` + `edge_circuit.record_err()` on cap pressure.
4. `record_blacklist_pending_overflow()` only when `_bl_ver == 0` (cold boot before first sync).

### Holdouts

`tests/bl_pending_cap_test.lua`, `bl_pending_byte_cap_bl_pending_unbounded`, `bl_pending_overflow_immediate_stamp_pending_overflow_immediate_stamp`.

---

## tarpit_delay_cap: Tarpit worker slot exhaustion — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was, when enabled) |
| Files | `edge-tarpit.lua`, `edge-metrics.lua` |
| Phase | access |

### Fix (2026-09-06)

1. Hard cap `EDGE_TARPIT_MAX_SEC` at 2 s (removed 15 s operator footgun).
2. `EDGE_TARPIT_MAX_CONCURRENT` (default 32): skip `ngx.sleep` when `tarpit_active` SHM counter exceeded; metric `tarpit_admit_reject_total`.

Default: tarpit **disabled**.

### Holdouts

`tests/tarpit_test.lua`, `tarpit_delay_cap_tarpit_delay_at_scale`.

---

## route_gate_env_cache: Hot-path `os.getenv` / per-request `io.open` — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-route-gate.lua`, `edge_track_policy.lua` |
| Phase | access |

### Fix (2026-09-06)

1. `edge-route-gate`: `EDGE_EXPOSE_*` cached at module load via `reload_env_flags()`.
2. `edge_track_policy`: `BODY_MODE`, `INGRESS_SCHEMA`, `EDGE_MAX_BODY` read once at module init (no `io.open` in `_M.run()`).

### Holdouts

`tests/route_gate_env_test.lua`, `route_gate_env_cache_route_gate_env_fallback`.

---

## negative_content_length_reject: Negative `Content-Length` not rejected at edge — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-parse-dfa.lua`, `edge_track_policy.lua` |
| Phase | access |

### Fix (2026-09-06)

`check_content_length(-1)` returns `ERR_MALFORMED`; `extract_campaign_id` rejects negative CL; `check_edge_limits` returns HTTP 400.

### Holdout

`negative_content_length_reject_negative_content_length`.

---

## http_get_json_body_cap: `edge-net.http_get_json` unbounded HTTP body read — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge-net.lua` |
| Phase | worker timer |

### Fix (2026-09-06)

`receive(MAX_HTTP_RESPONSE_BYTES + 1)` with caps `MAX_HTTP_RESPONSE_BYTES=69632`, `MAX_HTTP_BODY_BYTES=65536` before `cjson.decode`.

### Holdout

`http_get_json_body_cap_http_get_json_no_body_cap`.

---

## slot_map_before_node_weights: Node weights override slot-map shard affinity — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-shard-balancer.lua` |
| Phase | balance |

### Fix (2026-09-06)

Peer selection: `slot_map.get_shard()` index first; `node_weights.pick_peer_index()` only when shard nil. Preserves CRC32C campaign affinity when slot map active.

### Holdout

`weights_preempt_slot`.

---

## safe_page_campaign_id_normalize: Safe-page passes raw `campaign_id` to subrequest — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-safe-page.lua` |
| Phase | body_filter |

### Fix (2026-09-06)

`normalize_campaign_id()` via `edge-uuid.normalize()` before `/safe_page_content` subrequest; invalid ids become empty string.

### Holdouts

`tests/safe_page_cid_test.lua`, `safe_page_campaign_id_normalize_safe_page_unvalidated_cid`.

---

## blacklist_ip_canonical: Blacklist SHM key without IP canonicalization — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `edge-ip.lua`, `access-check.lua`, `edge-blacklist-sync.lua`, `edge-rl.lua` |
| Phase | access, worker sync |

### Issue (was)

`b:{ngx.var.remote_addr}` and `stamp_ips` used raw nginx IP strings. IPv4-mapped forms (`::ffff:1.2.3.4`) missed stamps keyed as `1.2.3.4` from Redis/XDP (split-brain L7 pass vs L4 drop).

### Fix (2026-09-06)

`edge-ip.canonical()` via libc `inet_pton`/`inet_ntop`: IPv4-mapped unmaps to dotted quad; native IPv6 uses `inet_ntop` canonical form. Applied on perimeter lookup, `stamp_ips`, and `edge_rl` ip: subject keys.

### Holdouts

`tests/ip_canonical_test.lua`, `blacklist_ip_canonical_blacklist_canonical_v4mapped`.

---

## campaign_id_scan_evasion: Per-campaign RL bypass when campaign_id beyond scan window — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `edge-campaign-id.lua`, `edge_track_policy.lua`, `edge-metrics.lua` |
| Phase | access (`edge_track_policy`) |

### Issue (was)

DFA returns `nil, nil` when `campaign_id` sits beyond `MAX_SCAN_BYTES` (8192). Policy applied IP-only `edge_rl` and proxied without `ngx.ctx.campaign_id`, evading per-campaign rate limits and shard affinity.

### Fix (2026-09-06)

`edge_campaign_id.scan_evasion()` detects scan-cap / CL-cap bodies with no resolved id (trusted header still allowed via `resolve_campaign_id`). `edge_track_policy` returns **400** + `campaign_id_scan_reject_total` metric instead of IP-only RL pass.

Small bodies without `campaign_id` still use IP-only RL fallback (nil_campaign_id_ip_rl_fallback preserved).

### Holdouts

`tests/campaign_id_scan_test.lua`, `campaign_id_scan_evasion_scan_cap_nil_campaign_id_reject`.

---

## bl_pending_overflow_immediate_stamp: `_bl_pending` overflow silently dropped quarantine IPs — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `edge-blacklist-sync.lua`, `edge-metrics.lua` |
| Phase | worker timer (blacklist incremental) |

### Issue (was)

bl_pending_byte_cap capped `_bl_pending` byte size but overflow IPs were counted and dropped without `b:{ip}` stamp until the next full `sync()` (~5 s). Fresh quarantine IPs could pass L7 perimeter during burst.

### Fix (2026-09-06)

`append_pending_ips()` calls `stamp_one_ip()` at current `_bl_ver` when the pending queue is full. Canonical IP keys via `edge-ip.lua`. Metric `blacklist_pending_immediate_stamp_total`.

### Holdouts

`tests/bl_pending_cap_test.lua`, `bl_pending_overflow_immediate_stamp_pending_overflow_immediate_stamp`, `bl_pending_byte_cap_bl_pending_unbounded`.

---

## pending_drain_restore_on_fail: `drain_pending_changelog` delete-before-stamp — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge-blacklist-sync.lua` |
| Phase | worker timer |

### Issue (was)

`drain_pending_changelog()` deleted `_bl_pending` before `stamp_ips()`. When `stamp_ips` returned false (Redis/connect failure path), the pending queue was lost.

### Fix (2026-09-06)

Snapshot `_bl_pending` before drain clear; restore snapshot when `stamp_ips(..., false)` fails. Success path unchanged (deferred tail re-queued via `append_pending_ips`).

### Holdouts

`tests/bl_pending_drain_test.lua`, `pending_drain_restore_on_fail`.

---

## incremental_quarantine_lag: Incremental quarantine propagation lag — **ACCEPTED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (residual) |
| Files | `edge-quarantine-sub.lua`, `edge-blacklist-sync.lua` |
| Phase | worker 0 pub/sub |

### Issue

Between Redis `blacklist:fraud` SADD and L7 `b:{ip}` stamp, requests can pass perimeter (no key yet). Incremental path uses `stamp_ips(..., false)` without bumping `_bl_ver` (by design; stale stamps from prior generation remain valid for unblocked IPs).

### Mitigation (no code change)

1. `fraud:quarantine` pub/sub -> `apply_quarantine_message` -> immediate `stamp_ips` on worker 0 thread.
2. Full `sync()` SMEMBERS backstop on timer (~5 s).
3. bl_pending_overflow_immediate_stamp immediate-stamp on `_bl_pending` overflow closes burst gap after stamp path is active.

### Operator signal

Monitor `ad_event_processor_edge_sync_last_success_timestamp` and quarantine-to-block funnel lag in ClickHouse; alert on sustained gap, not single-digit ms pub/sub delay.

---

## malformed_dfa_no_proxy: DFA `ERR_MALFORMED` proxied to tracker — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge_track_policy.lua`, `edge-metrics.lua` |
| Phase | access |

### Fix (2026-09-06)

`handle_parse_error()` rejects **400** + `parse_malformed_total` when DFA returns `ERR_MALFORMED` inside scan window (same gate as `ERR_OVERSIZE` -> 413).

### Holdouts

`tests/track_policy_failclosed_test.lua`, `apply_parse_error_gate(ERR_MALFORMED)`.

---

## read_body_fail_closed: `read_body` / peek failure fail-open — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge_track_policy.lua`, `edge-metrics.lua` |
| Phase | access |

### Fix (2026-09-06)

`read_bounded_body` and cosocket peek paths call `reject_body_unavailable()` (**400** + `body_read_failed_total`) instead of proxying with IP-only RL.

### Holdouts

`tests/track_policy_failclosed_test.lua` (`run_full` with failing `read_body`).

---

## options_track_rate_limit: `OPTIONS /track` skips fraud tier and edge_rl — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `access-check.lua`, `edge_track_policy.lua` |
| Phase | access |

### Fix (2026-09-06)

`edge_track_policy.run_options_track()` runs fraud tier + IP-only `edge_rl` before upstream CORS 204.

### Holdouts

`tests/track_policy_failclosed_test.lua` (`run_options_track` RL deny on second call).

---

## stream_mode_trusted_header: `EDGE_BODY_MODE=stream` without trusted header — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `edge_track_policy.lua` |
| Phase | access |

### Fix (2026-09-06)

`run_stream()` requires trusted `X-Edge-Trusted-Campaign-Id` (or configured header); missing id -> **400** `campaign_id_scan_reject_total` (same as campaign_id_scan_evasion scan evasion). Production default remains `EDGE_BODY_MODE=full`.

### Holdouts

`tests/track_policy_failclosed_test.lua` (`run_stream` without trusted header).

---

## Go tracker / filter

### lua_tier_degraded_fail_closed — Lua `tier_degraded` (code 20) accept without pacing/fcap — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `internal/filter/unified/unified_budget.go`, `unified-filter.lua` |
| Phase | tracker FilterEngine / UnifiedFilter |

### Fix (2026-09-06)

Go on Lua code 20: increment `filter_tier_degraded_total`, `RollbackDebit`, return `ErrFilterTimeout` (fail-closed). Lua still skips pacing/fcap/TTC under deadline pressure but accept+debit no longer reaches ingest.

### Holdouts

`TestUnifiedFilter_TierDegradationNearDeadline`, `TestUnifiedFilter_TierDegraded_failClosed_holdout`.

---

### click_cidr_table_nil_fail_closed — `/click` CIDR L1 table nil/unpublished fail-open — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `internal/ingest/landing_bundle.go` |
| Phase | gnet `/click` review traffic |

### Fix (2026-09-06)

When `CIDRBlockEnabled` and CIDR table nil or `!Ready()` → safe-view L1 before FilterEngine (fail-closed).

### Holdouts

`TestClickRedirect_CIDRBlockTableNil_FailClosed`, `TestClickRedirect_CIDRBlockTableNil_failOpen_holdout`.

---

### bpf_violations_ringbuf_required — edge-bpf-sync violations ringbuf open fail warn-only — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **High** (was) |
| Files | `cmd/edge-bpf-sync/main.go`, `doc.go` |
| Phase | XDP autoban sidecar |

### Fix (2026-09-06)

Violations pinned map + ringbuf reader required at startup (`os.Exit 1` on failure). Fingerprints ringbuf remains optional (warn, continue).

---

### dc_asn_always_checked — DC ASN check sampled ~1/8 — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `internal/filter/engine.go` |
| Phase | FraudFilter / FilterEngine |

### Fix (2026-09-06)

Removed `dcASNCheckSampleMask` sampling; `checkDCASN` runs on every check when table ready.

### Holdouts

`TestFraudFilter_DCASN_alwaysChecks_holdout`, `TestFraudFilter_DCASN_holdout`.

---

### l2_shadow_no_budget_debit — L2 shadow accept without budget debit — **ACCEPTED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (by design) |
| Files | `internal/filter/engine.go`, `filter_shadow_budget_test.go` |
| Phase | FraudFilter L2 shadow layer |

Shadow events log fraud signals and skip unified Lua debit by design. Holdout `TestFilterEngine_shadowSkipsUnifiedBudgetDebit_holdout` guards regression. Analytics must filter on `ShadowEvent` / L2 layer, not treat as billable accept.

---

### ttc_missing_impression_ts_fail_closed — TTC bypass Lua code 10 missing impression ts — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `internal/filter/unified/unified_budget.go` |
| Phase | UnifiedFilter Lua |

### Fix (2026-09-06)

When `TTCMin` enabled and Lua returns code 10: add `FraudReasonMissingImpTS`, rollback debit, return `ErrFilterTimeout`. Fail-open TTC bypass only when campaign TTC min is zero.

---

### click_l1_table_nil_fail_closed — L1 proxy/VPN, TLS fingerprint, IPv4/IPv6 rotation tables nil fail-open — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Medium** (was) |
| Files | `internal/ingest/landing_bundle.go` |
| Phase | gnet `/click` L1 hooks |

### Fix (2026-09-06)

When campaign flag enabled (`ProxyVPNBlockEnabled`, `TLSFingerprintBlockEnabled`, `CIDRBlockEnabled` for rotation) and feed table nil/unpublished → safe-view fail-closed (same pattern as click_cidr_table_nil_fail_closed CIDR).

### Holdouts

`TestClickRedirect_ProxyVPNBlockTableNil_FailClosed`; CIDR/rotation tests in `landing_cidr_block_hook_test.go`.

---

## circuit_breaker_min_samples — Circuit breaker below 100 samples fail-open — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-circuit.lua`, `access-check.lua` |
| Phase | access |

### Fix (2026-09-06)

`open()` opens when `total >= MIN_ERR_OPEN` (10) and err rate > 0.95 even before `SAMPLE_WINDOW` 100.

### Holdouts

`tests/circuit_breaker_test.lua` (`circuit_breaker_min_samples: 100% errs opens before SAMPLE_WINDOW`).

---

## fraud_score_suspect_without_trusted — Missing trusted fraud score pass tier — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-fraud-tier.lua` |
| Phase | access |

### Fix (2026-09-06)

`score_for_edge_rl()` returns `PASS_MAX+1` when trusted header absent → suspect tier RL (50% default), not full pass. Client `X-Fraud-Score` still cleared (client_fraud_score_rl_tier).

### Holdouts

`tests/fraud_score_trust_test.lua`, `edge_security_holdout_test.lua` fraud_score_suspect_without_trusted case.

---

## asn_whitelist_sync_failclosed — Redis config sync fail retains stale ASN whitelist — **REMEDIATED**

| Field | Detail |
| :--- | :--- |
| Severity | **Low** (was) |
| Files | `edge-config.lua` |
| Phase | worker timer |

### Fix (2026-09-06)

On Redis connect or HMGET failure: `invalidate_asn_whitelist()` bumps `_asn_ver` without new stamps → all prior `asn_cdn:*` / `asn_mobile:*` entries fail closed until next successful sync.

### Holdouts

`tests/asn_sync_failclosed_test.lua`.

---

## Remediated in `edge-parse-dfa.lua` (2026-09-06)

| Issue | Fix | Verified by |
| :--- | :--- | :--- |
| JSON escape `pos+=2` mis-sync | `advance_json_string` | `json_escape_odd_backslashes` |
| Truncation → false `ERR_MALFORMED` | `nil, nil` on scan limit | `openrtb_truncated_mid_parse`, `campaign_id_after_scan_budget` |
| `format_campaign_id` alloc storm | FFI `UUID_BUF` | `binary_uuid_normalized` |
| `os.getenv` per request in DFA | `DEFAULT_INGRESS_SCHEMA` at load | module init |
| OpenRTB `walk_object` stack overflow | `MAX_JSON_DEPTH = 32` | `json_depth_bomb` |
| Native JSON depth bomb | `MAX_NATIVE_JSON_DEPTH = 16` | `native_json_depth_bomb` |

---

## Coverage matrix

| Module | Audit slugs | Unit tests |
| :--- | :--- | :--- |
| access-check.lua | client_asn_blacklist_bypass, blacklist_ip_canonical, options_track_rate_limit | security holdout, track_policy_failclosed_test |
| edge-asn.lua | client_asn_blacklist_bypass | edge_config_test (partial) |
| edge-rl.lua | 02, nil_campaign_id_ip_rl_fallback, blacklist_ip_canonical | security holdout |
| edge_track_policy.lua | 02, header_campaign_id_trusted_only, peek_socket_fail_policy, native_json_depth_limit, bl_pending_byte_cap, tarpit_delay_cap, route_gate_env_cache, negative_content_length_reject, campaign_id_scan_evasion, malformed_dfa_no_proxy, read_body_fail_closed, stream_mode_trusted_header | track_policy_failclosed_test, campaign_id_scan_test |
| edge-campaign-id.lua | header_campaign_id_trusted_only, campaign_id_scan_evasion | campaign_id_trust_test, campaign_id_scan_test |
| edge-parse-dfa.lua | native_json_depth_limit + remediated | fault + security |
| edge-blacklist-sync.lua | bl_pending_byte_cap, blacklist_ip_canonical, bl_pending_overflow_immediate_stamp, pending_drain_restore_on_fail | bl_pending_cap_test, bl_pending_drain_test, blacklist_sync_test |
| edge-tarpit.lua | tarpit_delay_cap | tarpit_test + security |
| edge-route-gate.lua | route_gate_env_cache | security holdout |
| edge-net.lua | http_get_json_body_cap | security holdout |
| edge-shard-balancer.lua | slot_map_before_node_weights | security holdout |
| edge-safe-page.lua | safe_page_campaign_id_normalize | security holdout |

---

## Status

All edge access findings and blacklist_ip_canonical through asn_whitelist_sync_failclosed remediated or accepted (2026-09-06).  
Go tracker audit items remediated or accepted (2026-09-06).

Operator follow-ups:

1. Inject trusted headers at CDN inner hop (`X-Edge-Trusted-ASN`, `X-Edge-Trusted-Fraud-Score`, `X-Edge-Trusted-Campaign-Id`) when nginx is client-reachable.
2. Monitor `ad_event_processor_edge_blacklist_pending_immediate_stamp_total`, `ad_event_processor_edge_blacklist_pending_overflow_total`, and `tarpit_admit_reject_total` after deploy.  
