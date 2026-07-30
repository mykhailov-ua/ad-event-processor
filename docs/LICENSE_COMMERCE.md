# Vendor license commerce (Layer V only)

How **eSPX (vendor)** defines sellable SKUs, prices, and issues licenses. **Not** visible or editable on customer installs.

Customer-facing install policy: [SELF_HOSTED.md](./SELF_HOSTED.md).  
Protection overview (vendor IP + operator data + trust): [PROTECTION.md](./PROTECTION.md).

---

## Principles

1. **Signed JWT is the only authority** on customer hardware (`license.jwt` → `billing.license_status`).
2. **Customer cannot change** plan, features, or price locally — only replace file via vendor heartbeat.
3. **No mandatory traffic metering to vendor** — monthly license is time + feature flags, not PU/event overage to vendor. **Optional** opt-in aggregates are a separate channel ([TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md)).
4. **SKU constructor** lives on vendor side: YAML and/or vendor admin UI → issues JWT.

---

## SKU definition (declarative YAML — target GAP-PROD-03)

Vendor file example (`commercial/skus.yaml` — **not** shipped inside customer binary):

```yaml
skus:
  - code: ingest_pro
    display_name: Ingest + Antifraud Pro
    price_usd_monthly: 1200
    valid_days: 30                    # monthly period; renewed each heartbeat cycle
    grace_days: 7                     # after valid_until before EXPIRED
    offline_grace_days: 7             # heartbeat unreachable (GAP-PROD-04)
    heartbeat_interval_hours: 24
    pre_renewal_warn_days: 5          # SPA banner before month end
    features:
      rtb_live: false
      ebpf_xdp_edge: true
      ivt_ml_detector: false
      ml_fraud_boost: false
      multi_region: false
    limits:                          # install-wide caps (optional hard stops)
      max_regions: 1
      max_api_keys: 99
    bind:
      mode: fingerprint              # node-locked

  - code: network_enterprise
    display_name: Full network stack
    price_usd_monthly: 4800
    valid_days: 30
    features:
      rtb_live: true
      ivt_ml_detector: true
      ml_fraud_boost: true
      multi_region: true
```

**Signing pipeline:** vendor tool reads SKU + sale record → `vendor.licenses` row → Ed25519 JWT → customer download or heartbeat response.

---

## Vendor admin (existing schema direction)

Tables in `billing` schema on **license server** (not customer PG):

- `vendor.licenses` — `license_key`, `plan_code`, `valid_until`, `grace_days`, `limits_json`, `features_json`, `revoked`
- Heartbeat / activate APIs: `internal/licensing/client.go`

Customer install stores mirror in `billing.license_status` only.

---

## What customer install must NOT have

- SKU price list edit UI
- Unsigned `limits_json` override for Layer V
- Vendor `subscription_plans` seeds presented as "your bill from ESPX"

Operator tiers (`billing.subscription_plans` on customer PG) are Layer O — separate runbook.

---

## Heartbeat contract (target)

| Parameter | Env / claim | Behavior |
| :--- | :--- | :--- |
| Interval **X** | `ESPX_LICENSE_REFRESH_INTERVAL` | Try online refresh |
| Offline **Y** | JWT claim `offline_grace_days` (new) or server policy | Warn in SPA + metrics; ingest continues while cached JWT valid |
| Expiry grace | `grace_days` in JWT | After `valid_until`, GRACE state then EXPIRED |
| Revoked | Server `revoked=true` | Next heartbeat returns revoked token → hard stop |

**Today:** `GraceDays` applies after `valid_until` only; heartbeat failure falls back to file. GAP-PROD-04 adds explicit offline grace + UI warning.

---

## Anti-tamper and gray-market risk

Closed source limits funnel; open source in a gray niche is easy to fork without payment. Strategy is **layered** — no single measure is sufficient.

### Layer 1 — Cryptographic license (implemented)

| Control | Purpose |
| :--- | :--- |
| Ed25519 JWT | Customer binary embeds **public key only**; forging claims without private key fails |
| `VerifyJWT` on refresh | Tampered `license.jwt` → `EXPIRED` |
| `vendor.licenses.revoked` | Kill switch on next heartbeat |

### Layer 2 — Binding and activation

| Control | Purpose |
| :--- | :--- |
| `deployment_id` + `bind.fingerprint` | License not copy-paste to second datacenter without re-activation |
| Activate once per sale | License server records first fingerprint; anomaly on mismatch |
| Max activations per key | Floating vs node-locked SKUs in vendor YAML |

### Layer 3 — Online refresh (GAP-PROD-04)

| Control | Purpose |
| :--- | :--- |
| Heartbeat every **X** h | Stolen static JWT expires from server-side rotation |
| Offline grace **Y** days | Legitimate air-gap / outage; then warn → stop |
| Short JWT TTL on online path | Server returns 24–72h token; refresh extends |

Static file-only license with 1-year JWT is crackable once; **rotation** forces contact with license server.

### Layer 4 — Commercial (not technical)

| Control | Purpose |
| :--- | :--- |
| EULA + no redistribution | Deterrent even when courts are weak |
| Support tied to heartbeat | Updates/models only for active licenses |
| Named buyer watermark | `customer_name` in JWT for leak tracing |

### Layer 5 — What not to do

| Anti-pattern | Why |
| :--- | :--- |
| **Mandatory** phone-home traffic metrics | Violates self-hosted trust; customers disable or fork |
| Mixing telemetry into license heartbeat without schema | Conflates billing trust with analytics; triggers связки fear |
| Obfuscation-only protection | Reversible; wastes engineering |
| GPL/fully open **Pro** core in gray niche | Fork without paying; funnel collapses |
| Per-event **vendor** license enforcement | Contradicts unlimited volume on customer hardware |

Opt-in product telemetry and threat intel are allowed — see [TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md).

### Layer 6 — Distribution model (recommended)

```text
Public:  docs, OpenAPI, integration guides
Binary:  stripped binaries per OS/arch (no source)
Source:  separate enterprise escrow / audit — not public GitHub for core tracker
Models:  download from vendor CDN with license cookie (ML SKUs)
```

**Funnel trade-off:** binary + trial license lowers friction vs source; serious buyers get audit under NDA. Gray operators who never paid are not the ICP — optimize for networks that need support and updates.

### Layer 7 — Detection (vendor server)

| Signal | Action |
| :--- | :--- |
| Same `license_key` from many fingerprints | Flag / revoke |
| Heartbeat from conflicting regions simultaneously | Clone suspicion |
| Impossible version strings | Block refresh |

No PII/traffic content on license heartbeat — only deployment metadata. Optional telemetry is a **separate** opt-in channel.

---

## Sales matrix (reference)

| SKU | Typical buyer | Compose profile |
| :--- | :--- | :--- |
| `ingest_pro` | Arbitrage team | `ingest_only` |
| `network_std` | Small ad network | `network_operator` |
| `network_enterprise` | Network + RTB + ML | `network_operator` + `analytics_ml` |
