# Protection model (self-hosted eSPX)

Unified policy for **vendor IP protection**, **operator data protection**, and **mutual trust** on customer bare metal. This document summarizes decisions from product discussions; detail lives in linked runbooks.

| Audience | Start here |
| :--- | :--- |
| **Operator (install owner)** | § Operator data + § Trust controls |
| **Vendor (eSPX sales/engineering)** | § Vendor IP + § Distribution |
| **Security review** | All sections + linked runbooks |

**Detail documents:**

- [runbooks/DATA_SECURITY.md](./runbooks/DATA_SECURITY.md) — encryption, secrets, retention on operator hardware
- [runbooks/RECONCILIATION_AND_SETTLEMENT.md](./runbooks/RECONCILIATION_AND_SETTLEMENT.md) — background workers, PG/CH drift audit, money flow
- [runbooks/RBAC_AND_PROTECTION.md](./runbooks/RBAC_AND_PROTECTION.md) — internal roles, field masking, staff access control
- [TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md) — egress channels, opt-in telemetry, collective antifraud
- [LICENSE_COMMERCE.md](./LICENSE_COMMERCE.md) — monthly license, anti-tamper, SKU signing

---

## 1. Three protection goals (not one)

```text
┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐
│ A. Vendor IP        │   │ B. Operator data    │   │ C. Mutual trust     │
│ License, binary,    │   │ Events, ledger,     │   │ No связки leak;     │
│ models, updates     │   │ keys on their disk  │   │ auditable egress    │
└─────────────────────┘   └─────────────────────┘   └─────────────────────┘
     LICENSE_COMMERCE          DATA_SECURITY           TELEMETRY_AND_TRUST
```

Optimizing only (A) increases operator fear. Optimizing only (B) without (A) invites gray-market forks. **(C)** is the architecture that lets (A) and (B) coexist.

---

## 2. Protection matrix

| Risk | Who loses | Primary controls |
| :--- | :--- | :--- |
| Stolen license / cracked binary | Vendor | Ed25519 JWT, heartbeat rotation, fingerprint bind, revocation |
| Stolen disk / backup | Operator | LUKS, encrypted backups, PG retention |
| Network sniffing | Operator | TLS edge + internal TLS (`sslmode=verify-full`) |
| Insider abuse | Operator | RBAC, `admin_audit_log`, least privilege |
| Vendor steals связки | Operator | Separate egress schemas; telemetry **opt-in off** by default; allowlist |
| Competitor copies traffic intel | Operator | Threat intel: hashes only, no raw events in channel 3 |
| CH analytics dump | Operator | `pkg/piihash` — no raw IP/UA in CH (GAP-DATA-01) |
| PG events dump | Operator | Plaintext IP/UA today; encrypted disk + retention (GAP-DATA-02) |
| Svyazka theft by staff | Operator | RBAC roles, field-level masking (URL/creatives), audit log |
| Mandatory vendor metering | Operator | **No** per-event license billing; unlimited on own hardware |
| Fork without paying | Vendor | Closed Pro binary; updates/models via active license |

---

## 3. Vendor IP protection (Layer V)

### Monthly license (not annual-first)

- Subscription renews via license server → short-lived signed JWT.
- Payment failure: **offline grace Y days** + SPA warning before `license_expired` on ingest (GAP-PROD-04).
- Proactive refresh **5–7 days before** `valid_until`.

### Cryptographic controls (implemented)

| Control | Location |
| :--- | :--- |
| Ed25519-signed `license.jwt` | `internal/licensing/verify.go` |
| Public key only in customer binary | Private key on vendor license server |
| `billing.license_status` mirror | Cold-path module gates |
| Revocation | `vendor.licenses.revoked` + next heartbeat |

### Binding and anti-clone (target hardening — GAP-PROD-06)

| Control | Purpose |
| :--- | :--- |
| `deployment_id` + `bind.fingerprint` | No paste license to second site without re-activate |
| Max activations per `license_key` | Node-locked vs floating SKU |
| JWT rotation on heartbeat | Stolen static file expires |
| Clone detection on server | Same key, many fingerprints / regions → flag/revoke |

### What does **not** protect the vendor

| Measure | Why insufficient alone |
| :--- | :--- |
| Code obfuscation | Reversible |
| Long-lived JWT file | Crack once, run forever |
| Per-event phone-home | Violates self-hosted trust; patched out |

**Sustainable lever:** active license required for **updates**, ML model packs, and threat intel feed — value worth paying for.

### Gray market / closed source

| Topic | Policy |
| :--- | :--- |
| Niche | Gray ad-tech; legal enforcement weak |
| Pro core | **Closed binary** — no public source for ingest/RTB/antifraud |
| Open portfolio (current contractor stage) | Public GitHub as **engineering proof** — acceptable before first product sales |
| Future split | Community (docs, demos) vs Pro (GAP-PROD-10) when paying customers appear |
| Enterprise trust | Source escrow / read-only audit under NDA — not public repo |

Full layer list: [LICENSE_COMMERCE.md § Anti-tamper](./LICENSE_COMMERCE.md#anti-tamper-and-gray-market-risk).

---

## 4. Operator data protection (Layer O — on their hardware)

The vendor **does not host** operator traffic. Protection on customer metal is **shared responsibility**:

| Layer | Owner | Actions |
| :--- | :--- | :--- |
| Disk encryption | Operator | LUKS on PG, Redis, CH, spool volumes |
| TLS | Operator + product | Edge in product; internal TLS in prod profile (GAP-DATA-02) |
| Secrets | Operator | Vault; unique `PII_SALT_HEX`; never commit `.env` |
| Minimization | Product + operator | CH hashes; PG retention policy |
| Access | Product + operator | RBAC, API key scoping, audit log |

### In-product today

- TLS at nginx edge.
- Argon2id + PASETO for admin auth.
- HighwayHash PII fields in ClickHouse (`PII_SALT_HEX`, `PII_SALT_VERSION`).
- HMAC on TCP control plane.
- Edge XDP / Lua rate limit and blocklist.

### Gaps (documented, not hidden)

- PostgreSQL `events.ip_address`, `user_agent` — **plaintext** (needed for settlement/dedup today).
- No application-level disk encryption.
- Reference compose uses plaintext internal network for dev.

**Minimum viable security checklist:** [runbooks/DATA_SECURITY.md § MVSS](./runbooks/DATA_SECURITY.md#minimum-viable-security-sell-sheet--install-gate).

---

## 5. Mutual trust — protecting against связки leak

Operators fear that a **closed binary** exfiltrates campaigns, domains, or sources. Technical response:

### Rule: three separate outbound channels

| Channel | Required? | May contain |
| :--- | :---: | :--- |
| **1. License heartbeat** | Yes (monthly sub) | `deployment_id`, fingerprint, version, uptime |
| **2. Product telemetry** | No (default off) | Install-wide aggregates only |
| **3. Threat intel** | No (separate opt-in) | Reject histograms, signal **hashes** |

**Never in any channel:** `campaign_id`, domain, URL, referrer, `click_id`, raw IP, creatives, payout.

### Operator-verifiable controls

| Control | Effect |
| :--- | :--- |
| `ESPX_TELEMETRY_OPT_IN=0` | No channel 2 |
| Threat intel toggle off | No channel 3 |
| Egress firewall allowlist | Only `license.<vendor>` (and optional telemetry host) |
| Proxy with body logging | Operator audits JSON before it leaves |
| Published JSON Schema | Per-channel field allowlist |
| Air-gap license file | Heartbeat optional profile; no vendor egress except manual renewal |

### Encryption ≠ vendor trust

LUKS protects against **third parties** (hoster, disk theft). It does **not** prove the binary has no backdoor. Trust for egress is **schema + opt-in + audit**, not encryption alone.

Detail: [TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md).

---

## 6. Collective antifraud (optional value exchange)

Without operator traffic on vendor servers, **fleet-wide ML** requires opt-in channel 3:

| Operator sends | Operator receives |
| :--- | :--- |
| Hourly reject rates by `filter_kind` | Updated L3 / IP blocklists |
| Hashed signal classes (not raw clicks) | `ml:score:boost` model packs via outbox |
| No campaigns or URLs | Anomaly bulletins (aggregated) |

**Sales line:** immunization for the tracker, not surveillance of связки.

Participation is **not** required for base monthly license.

---

## 7. Deploy profiles and attack surface

| Profile | Exposure | Protection focus |
| :--- | :--- | :--- |
| `ingest_only` | Smaller stack; no payment UI | Edge + PG + Redis; CH optional |
| `network_operator` | + Stripe/crypto, self-serve API | + PCI hygiene, webhook TLS, RBAC |
| `analytics_ml` | + ClickHouse, batch ML | + `PII_SALT_HEX`, CH disk encryption |

Smaller profile = fewer binaries to patch and fewer secrets to rotate.

---

## 8. Paranoid vs standard operator profiles

### Standard (recommended default)

- Online monthly heartbeat.
- Telemetry off.
- LUKS + firewall + `sslmode=verify-full`.
- Unique PII salt.

### Paranoid (high-sensitivity arbitrage)

- `ESPX_LICENSE_MODE=file` where contract allows; long offline grace documented.
- Telemetry and threat intel **off**.
- Egress deny-by-default; manual license file renewal.
- Internal network only; no public management API.
- Optional source escrow review under NDA.

### Enterprise network

- Standard + internal mTLS.
- Threat intel opt-in for blocklist feed.
- SOC2-style password policy (auth schema supports history).

---

## 9. Incident playbooks (summary)

| Event | Operator actions | Vendor role |
| :--- | :--- | :--- |
| Suspected egress leak | Capture proxy logs; compare to schema; disable telemetry | Publish schema diff; incident notice |
| Stolen backup | Rotate secrets; restore from encrypted backup | None (no customer data held) |
| License clone suspicion | — | Revoke key; investigate fingerprints |
| Compromised `PII_SALT_HEX` | Rotate salt + version; plan CH hash migration | Guidance in DATA_SECURITY runbook |
| Cracked binary circulating | — | Revoke + rotate signing keys; release patch |

Full operator steps: [runbooks/DATA_SECURITY.md § Incident response](./runbooks/DATA_SECURITY.md#incident-response-operator).

---

## 10. Open decisions (protection architecture)

| # | Topic | Doc |
| :---: | :--- | :--- |
| 1 | PG IP: retention vs hash-at-insert | DATA_SECURITY, GAP-DATA-02 |
| 2 | Production internal TLS defaults | GAP-DATA-02 |
| 3 | Heartbeat X / offline Y defaults | GAP-PROD-04 |
| 4 | Fingerprint mismatch: block vs warn | GAP-PROD-06 |
| 5 | Threat intel push vs pull | TELEMETRY_AND_TRUST |
| 6 | Community vs Pro source split timing | GAP-PROD-10 |
| 7 | Source escrow criteria | LICENSE_COMMERCE |

---

## 11. Engineering backlog (protection-related)

Full specs: [.cursor/GAP_SPECS.md](../.cursor/GAP_SPECS.md).

| ID | Topic |
| :---: | :--- |
| GAP-DATA-01 | PII hash before CH — **shipped** |
| GAP-DATA-02 | PG retention, prod TLS profile, MVSS checklist |
| GAP-HYG-30 | PG volume meter, recon worker, pinned settlement |
| GAP-PROD-04 | Monthly license offline grace + SPA warnings |
| GAP-PROD-06 | License bind enforcement, clone detection |
| GAP-PROD-08 | Opt-in telemetry pulse (channel 2) |
| GAP-PROD-10 | Community vs Pro distribution |
| GAP-PROD-11 | RBAC & field masking |
| GAP-OPS-05 | `espx doctor`, auto-tuning, debug bundle |

---

## 12. FAQ

**Does eSPX encrypt all customer data by default?**  
No. CH analytics uses hashed PII; PG events store IP/UA in plaintext today. Disk encryption is operator responsibility.

**Can the vendor see our RPS?**  
Only if the operator opts into channel 2. Marketing aggregates use opted-in installs only.

**Is closed source safe for us?**  
Safe for data on disk if operator follows DATA_SECURITY. Safe from vendor связки leak if telemetry is off and egress is audited.

**Why monthly license if we run on our hardware?**  
Payment for software **features, updates, and support** — not for hosting or metering your traffic.

**GitHub public repo — is that a security problem?**  
For **product sales**, Pro ingest core should move private (GAP-PROD-10). For **portfolio**, public repo demonstrates capability; it is not the shipped Pro binary.
