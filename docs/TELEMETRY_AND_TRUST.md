# Telemetry, trust, and product evolution (self-hosted)

How eSPX handles **vendor ↔ operator trust**, **opt-in telemetry**, **collective antifraud**, and **product development without owning customer traffic**. Complements [SELF_HOSTED.md](./SELF_HOSTED.md), [LICENSE_COMMERCE.md](./LICENSE_COMMERCE.md), and [PROTECTION.md](./PROTECTION.md).

---

## Context

eSPX is sold as **closed binaries** on **customer bare metal**. The vendor does not host impressions, campaigns, or landing URLs. By default the vendor has **no fleet data** — only what customers choose to send and what the **license heartbeat** carries.

This document records product policy from engineering discussions (2026-07).

---

## Commercial model

| Topic | Policy |
| :--- | :--- |
| **License period** | **Monthly** subscription (not annual-first); renewed via license server / JWT `valid_until` |
| **Vendor metering** | **No** per-event billing to vendor; customer runs unlimited volume on own hardware within licensed **features** |
| **Vendor revenue** | Monthly license fee + optional SKUs (RTB, ML, threat intel feed) — invoiced **outside** the install |
| **Payment failure** | Long offline grace **Y** days + SPA warning before `EXPIRED`; proactive refresh before month-end (GAP-PROD-04) |

Recommended monthly UX: heartbeat attempts refresh **5–7 days before** `valid_until`; failed payment must not kill ingest at 03:00 without warning window.

---

## Why operators fear "слив связок"

In ad-tech, a **связка** is the profitable combination of traffic source, geo, creative, landing, and payout. Leakage means competitors copy or cut the source.

Closed-source self-hosted amplifies paranoia: the operator cannot diff the binary, so any outbound call is suspect.

**Response is architectural separation**, not marketing promises alone.

---

## Three outbound channels (must stay separate)

```text
┌─────────────────────────────────────────────────────────────────┐
│ 1. License heartbeat (required for monthly subscription)         │
│    deployment_id, fingerprint, version, uptime → JWT refresh      │
│    NEVER: campaign_id, domain, URL, referrer, click_id, IP        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 2. Product telemetry (opt-in, default OFF)                       │
│    Install-wide aggregates for vendor marketing + fleet health   │
│    NEVER: per-campaign or per-source fields                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 3. Threat intel exchange (opt-in, separate SKU / toggle)         │
│    Anonymized reject stats + signal hashes → collective feed back  │
│    NEVER: raw events, creatives, domains                          │
└─────────────────────────────────────────────────────────────────┘
```

Operators can firewall so that **only (1)** is allowed. (2) and (3) are independent toggles in bundled SPA settings.

**Auditable:** publish JSON Schema for each channel; operator can log outbound bodies at their proxy.

---

## Channel 1 — License heartbeat (required)

**Purpose:** monthly renewal, revocation, JWT rotation, activation binding.

**Payload (today / target):** see `internal/licensing/client.go` — `license_key`, `deployment_id`, `fingerprint`, `version`, `uptime_seconds`.

**Not included:** traffic volume totals are **optional** and belong in channel 2, not heartbeat, unless explicitly documented in schema v2.

**Env:** `ESPX_LICENSE_MODE=online`, `ESPX_LICENSE_REFRESH_INTERVAL` (default 24h).

---

## Channel 2 — Product telemetry (opt-in)

**Purpose:** vendor marketing ("N deployments, aggregate throughput"), release adoption, SRE signals — **without** competitive intelligence on any single operator.

### Policy

| Rule | Detail |
| :--- | :--- |
| Default | `ESPX_TELEMETRY_OPT_IN=0` |
| Cadence | Hourly pulse (or daily minimum) |
| Aggregation | Counters over entire install, not per customer/advertiser |
| Marketing use | Only **sum across opted-in** deployments; disclose "based on N opt-in installs" |

### Example pulse payload (target — GAP-PROD-08)

```json
{
  "schema_version": 1,
  "deployment_id": "uuid",
  "binary_version": "1.2.3",
  "sku": "ingest_pro",
  "window_sec": 3600,
  "accepted_events": 1250000,
  "rejected_events": 42000,
  "reject_histogram": {
    "budget": 12000,
    "dedup": 18000,
    "geo": 5000,
    "fraud_tier_block": 7000
  },
  "peak_rps": 8500,
  "dc_region": "eu-central"
}
```

### Forbidden fields

`campaign_id`, `customer_id`, `domain`, `url`, `referrer`, `click_id`, `user_id`, raw IP/UA, creative IDs, payout, placement names.

### Implementation note

Local **atomic counters** on tracker/management; cold-path worker flushes once per window. No per-request HTTP to vendor.

---

## Channel 3 — Threat intel exchange (opt-in)

**Purpose:** sell collective antifraud value when vendor has **no** access to customer CH/PG.

### Operator → vendor (minimal)

Hourly aggregates only:

- `reject_rate` by `filter_kind` / fraud tier (%)
- optional **hashed signal classes** already used on hot path (e.g. normalized UA class, /24 prefix hash) — never reversible to a specific click

### Vendor → operator (incentive)

Pull or push via license server / outbox:

- updated L3 / IP range blocklists (community-sourced, k-anonymity thresholds)
- `ml:score:boost` snapshot packs (`ML_MODEL_VERSION` outbox pattern)
- anomaly bulletins: "dedup spike class X in region Y" — no customer names

### Sales pitch (operator-facing)

> You do not send связки or creatives. You send install-wide reject statistics and anonymized bot signal classes. In return you receive blocklists and model packs trained from the same opt-in pool — immunization for your tracker without vendor access to your traffic.

**License:** threat intel participation is **not** required for base monthly license.

---

## Trust controls (operator-verifiable)

| Control | What it gives |
| :--- | :--- |
| **Network allowlist** | Operator permits only `license.<vendor>` host; telemetry host optional second FQDN |
| **Published schemas** | `docs/schemas/telemetry-pulse.json`, heartbeat OpenAPI on license server |
| **Proxy logging** | Operator terminates TLS, inspects JSON body |
| **No telemetry = full function** | Ingest and license work; only intel feed and vendor aggregate stats missing |
| **Escrow / audit** | Enterprise: read-only source under NDA for core ingest (see [LICENSE_COMMERCE.md](./LICENSE_COMMERCE.md)) |
| **Air-gap profile** | `ESPX_LICENSE_MODE=file` + long offline grace; no channels 2–3 |

---

## Vendor has zero data today — how to evolve the product

Self-hosted binary without opt-in telemetry means the vendor sells an **engine**, not a data platform. Roadmap without customer PII:

| Source | Use |
| :--- | :--- |
| **In-product feedback form** (bundled SPA) | Bugs, feature votes, severity — no campaign logs |
| **Opt-in diagnostic bundle** | On-demand support pack: versions, redacted config, 1h aggregate reject rates |
| **Pilot customers under NDA** | Explicit threat-sharing agreement for first intel feed |
| **Public IVT / bot benchmarks** | Rules and models shipped in **releases** |
| **Synthetic load + chaos tests** | Regression for ingest SLA |
| **Release cadence** | Value = fixes, OpenRTB, edge — not "we learned from your data" |

Do not claim network-wide ML training until opt-in threat intel pool exists.

---

## In-product feedback (GAP-PROD-09)

Bundled SPA should include:

- type: bug | feature | support
- fields: contact email, `deployment_id`, `binary_version`, `sku` (auto-filled)
- optional: attach opt-in diagnostic bundle
- no free-text campaign URLs required

Vendor uses this for roadmap until telemetry cohort is large enough.

---

## Monthly license and grace

| Phase | Behavior |
| :--- | :--- |
| Active | `valid_until` in future → ingest allowed |
| Pre-expiry | SPA banner: "renew in N days"; heartbeat retries increase |
| Payment / refresh failed, within **Y** offline days | Warning metrics + SPA; ingest continues on cached JWT |
| After **Y** offline or JWT expired + `grace_days` | `license_expired` on hot path |

**Y** and heartbeat **X** are SKU parameters in vendor YAML ([LICENSE_COMMERCE.md](./LICENSE_COMMERCE.md)).

Telemetry failure must **never** block ingest or license state.

---

## GitHub public repo vs future commercial product

**Current stage (vendor as B2B contractor):** public GitHub is a **plus** — portfolio proof of gnet, zero-alloc ingest, Redis sharding, settlement design.

**Future product stage:**

| Public (Community) | Private / binary (Pro) |
| :--- | :--- |
| Architecture docs, benchmarks, OpenAPI | Full RTB live, advanced antifraud |
| Simplified ingest demo or subset | Collective threat intel client |
| Integration guides | Monthly license enforcement build |

Do not delete portfolio history in panic; **split** when first paying self-hosted customers appear (GAP-PROD-10).

---

## Community vs Pro (reference split)

| Capability | Community (public) | Pro (closed binary) |
| :--- | :---: | :---: |
| Core `/track` + Lua budget | partial / demo | full |
| RTB live | no | license-gated |
| ebpf edge | no | license-gated |
| IVT / ML cold path | no | license-gated |
| Operator payment/billing stack | docs only | full |
| Threat intel feed | no | opt-in SKU |
| Support | community | monthly license |

Exact boundary TBD at first sale; document in vendor SKU YAML.

---

## Business risks (documented)

| Risk | Mitigation |
| :--- | :--- |
| "You steal связки" | Three channels, schemas, opt-in, allowlist |
| **SRE overhead (PG+Redis+CH+6 binaries)** | Deploy profiles `ingest_only`, optional CH (GAP-PROD-05); [DATA_SECURITY runbook](./runbooks/DATA_SECURITY.md) MVSS |
| No UI | Bundled SPA (GAP-PROD-02) |
| Gray market cracks | Layered license + **updates worth paying for** (intel, models) |
| Zero vendor data | Honest positioning; feedback + opt-in intel path |
| Monthly license outage | Long Y grace + pre-expiry warnings |

---

## Engineering backlog

| ID | Task |
| :--- | :--- |
| **GAP-PROD-08** | Opt-in product telemetry: local counters, hourly pulse, `ESPX_TELEMETRY_OPT_IN`, published schema |
| **GAP-PROD-09** | SPA feedback form + optional diagnostic bundle |
| **GAP-PROD-10** | Community vs Pro repo/binary split policy doc + release process |
| **GAP-PROD-04** | Monthly license heartbeat, offline grace Y, SPA warnings |
| **GAP-PROD-06** | Fingerprint bind, activation limits |

---

## FAQ

**Can the vendor track total RPS across all customers for marketing?**  
Only from **opt-in** channel 2, aggregated and anonymized per install. Quote: "based on N opted-in deployments."

**Does opt-in telemetry help the operator?**  
Channel 2 is primarily vendor fleet health/marketing. **Channel 3** is the antifraud value exchange (blocklists, model packs).

**Is heartbeat the same as telemetry?**  
No. Heartbeat is required for license; telemetry is optional and schema-separated.

**Does the vendor see fraud on customer traffic by default?**  
No. Fraud signals stay on customer PG/CH/Redis unless operator opts into channel 3.
