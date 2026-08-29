# Antifraud reference

Operator reference for tracker fraud layers and cold-path workers. Code: `internal/ingest`, `internal/fraud`, `internal/edge`, `cmd/ivt-detector`, `cmd/fraud-scorer`. Architecture: [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md).

---

## Hot vs cold

| Path | Scope | Sync I/O |
| :--- | :--- | :--- |
| Hot | `/track`, `/click`, `/tg/*` | At most one Redis `EVALSHA` per accepted event (zero when local quanta full-skip). No Postgres, ClickHouse, or ML on request thread. |
| Cold | Control `:8188`, processor `:8186`, IVT/ML sidecars | Postgres, ClickHouse, outbox, batch ML. |

`/openrtb/bid` does not run the full `FilterEngine` chain.

SLA targets: handler p95 < 50 ms, p99 < 80 ms; unified-filter Lua p99 < 10 ms per shard (load test / Prometheus, not isolated microbenches).

ML runs only in `cmd/fraud-scorer`. Tracker reads `ml:score:boost:*` from an in-memory snapshot (`SettingsWatcher`).

---

## Layer decision

When fraud signals exist before `UnifiedFilter`:

| Outcome | UnifiedFilter / Lua | Main stream | Debit |
| :--- | :--- | :--- | :--- |
| None | Runs | Yes | Yes |
| L2 shadow | Skipped | Yes (`ShadowEvent=true`) | No |
| L1 reject | Skipped | No | No |

Fraud rejects also enqueue a separate Redis fraud stream (`FraudStreamWriter`).

---

## Reaction levels

| Level | HTTP (`silent_reject_enabled`) | Analytics |
| :--- | :--- | :--- |
| L1 reject | 403, or decoy 202/302 when flag on | Fraud stream; `silent_reject_event` when silent |
| L2 shadow | Normal accept | Main stream, shadow flag |
| L3 blacklist | Contributes to L1 | IP on `blacklist:fraud`; edge may 403 first |

Campaign flag: `silent_reject_enabled` (legacy JSON alias `ghost_ivt_enabled` on PATCH). ClickHouse column: `silent_reject_event`.

---

## Hot-path signals (summary)

| Code | Weight | Tier |
| :--- | ---: | :--- |
| `datacenter_ip`, `low_ttc`, `tls_blocklist`, `moderator_ip` | 45 | L1-high |
| `l3_blocklist` | 100 | L3 |
| `missing_imp_ts`, `device_mismatch`, `sec_fetch_anomaly`, `client_hints_mismatch`, TLS/H2 anomalies, `tcp_mss_anomaly`, `tcp_tunnel_mss`, `tcp_syn_os_mismatch`, `json_serialization_bot`, behavior telemetry, `ipv4_rotation`, `residential_proxy`, `attestation_missing`, … | 35 | L2-weak |

Full registry: `fraudReasonRegistry` in `filter_errors.go`. Presets: `conservative`, `balanced`, `aggressive`, `enhanced_defense`, `social_in_app`.

**CDN caveat:** TCP MSS, TTL, SYN signature, and client TLS fingerprints need edge headers (`X-TCP-MSS`, `X-TLS-JA3`, etc.). Without them, signals fail-open. Disable or expect degraded signals behind CDN/ALB.

---

## Redis and caches

| Check | Steady path |
| :--- | :--- |
| Placement blacklist | In-process TTL cache; `HEXISTS` on miss |
| Fraud blacklist | In-process TTL cache; `SISMEMBER` on miss |
| Ingress RPD | `EntitlementsFilter` — one `INCR` per event |
| ML boost | Go snapshot only; Redis sync async |

`LOCAL_QUOTA_MODE=live` can skip sync `EVALSHA` for eligible traffic.

---

## Edge and XDP (Enterprise)

| Component | Role |
| :--- | :--- |
| nginx Lua | Rate limit, blacklist cache, proxy to tracker |
| `edge-bpf-sync` | Redis → BPF maps |
| `edge-xdp` | L3/L4 drop at NIC |

XDP drops listed hosts and flood patterns — not app-layer residential fraud.

---

## Cold workers

Requires ClickHouse and compose profile `analytics-ml` unless noted.

| Worker | Role |
| :--- | :--- |
| `ivt-detector` | CH rules: CTR spikes, fingerprint clusters, interval bots, `rtt_split_tunnel`, mobile biometrics |
| `fraud-scorer` | Batch LightGBM; writes `ml:score:boost:{campaign_id}` |
| Processor | Conversion smart reject before payout (no click, low TTC, duplicate, IP drift) |

ML actions (`EnqueueFraudThreatBatch`): `boost`, `blacklist`, `silent_reject` (alias `ghost` on enqueue). `silent_reject` adds IP to blacklist — does **not** flip `silent_reject_enabled` on campaigns.

---

## CGNAT mobile policy

When mobile carrier ASN detected, skip IP-frequency signals only (`ipv4_rotation`, ingress RPD). Does not bypass datacenter, TLS blocklist, L3, attestation, or budget Lua.

Knobs: `cgnat_ip_policy_enabled` on campaign; `CGNAT_MOBILE_IP_BYPASS` env.

---

## Admin surfaces

- Campaign fraud: `PATCH /api/v1/campaigns/{id}/fraud`
- Reports: silent-reject funnel, layer desync, fraud breakdown, evidence pack (`fraud_dispute_evidence` SKU on Scale+)
- Ops: `/api/v1/ops/ml-model`, fraud labels/decisions/overrides

---

## Verification

```bash
go test ./internal/ingest/ -run 'Fraud|SafePage|TLS|SilentReject' -count=1
go test ./internal/edge/... -count=1
bash scripts/ci/naming/antifraud_doc.sh
```
