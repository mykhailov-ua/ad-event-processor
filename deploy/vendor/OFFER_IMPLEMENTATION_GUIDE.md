# Offer implementation guide (internal sales and ops)

**Audience:** sales managers, account managers, vendor ops.  
**Not for customers.** Pair with [PUBLIC_OFFER.md](./PUBLIC_OFFER.md), [SALES.md](./SALES.md), [INVOICE.md](./INVOICE.md), [KEYS.md](./KEYS.md), [MARKETING.md](./MARKETING.md).

**Product name in buyer comms:** **ad-event-processor**. Legacy engineering codenames are internal only.

---

## 1. Legal basis (what to tell legal / finance)

**Stripe / processor one-liner:** "License of self-hosted server software for HTTP traffic measurement, event ingestion, and campaign routing. Customer runs on own infrastructure; vendor does not store customer traffic data."

| Topic | Position |
| :--- | :--- |
| Contract type | Public offer + acceptance (CCU Arts. 633, 638, 641-642), electronic commerce (Law No. 675-V). |
| What we sell | **License** to self-hosted Software + delivery of artifacts and JWT. Not SaaS hosting. |
| When contract binds | Payment confirmed **or** pilot JWT issued after checkbox acceptance **or** paid JWT applied after invoice. |
| Data roles | Buyer is controller of traffic/campaign data on their VPS. We are **not** processor of that data in default model. We are controller of buyer contact/billing/trial-registry data only. |
| GDPR DPA | **Not required** for standard self-hosted delivery. Required only if we run managed ops that touch buyer personal data. |
| Dual-use / CC Art. 361 | Software is **lawful ad infrastructure** (like a knife or database — dual-use). **No feature designed to violate Art. 361.** We store no ops data. **Criminal liability = buyer-operator only** (Offer Sections 3.6, 9.1, 11.5, 12). |
| Liability cap | 3 months fees (B2B); consumers keep mandatory rights. |
| Refunds | B2B: no refund after JWT delivery. Pilot: free. |

**Before go-live:** replace all `[PLACEHOLDER]` fields in `PUBLIC_OFFER.md`, publish on website/checkout, and get Ukrainian counsel sign-off.

**Article numbers:** civil offer = **CCU 641-642**; criminal misuse by operator = **CC (KKU) 361** et seq.

**Dual-use pitch (legal):** We supply neutral infrastructure for lawful ad tracking on the buyer's own servers. We do not host data and do not ship penetration/exploit tooling. If a buyer commits Art. 361, that is the **buyer's** act on **their** deployment — not supplier liability for dual-use goods.

---

## 1.1 Dual-use positioning (use with legal/compliance questions)

| Claim | Wording |
| :--- | :--- |
| What we are | Vendor of **self-hosted ad event processing** binaries + offline license |
| What we are not | Hosted SaaS, data processor of traffic, exploit vendor, access-bypass tool |
| Data | **Zero custody** of buyer campaigns/events in standard model |
| Art. 361 KKU | Applies to **unauthorized interference** by the **operator**; product has **lawful design purpose**; no inherent Art. 361 functionality |
| Analogy | General tool (server stack / knife): supplier not criminally liable for buyer's misuse when supply is for lawful trade |
| Contract | PUBLIC_OFFER Sections **3.6** (dual-use), **9.1** (operator liability), **11.5** (no vendor criminal liability), **12** (indemnity) |

Do not promise "immunity from prosecution" — say **liability allocation** and **factual architecture** (no data, no unlawful features).

## 2. Sales funnel and acceptance mechanics

### 2.1 Pilot (free)

1. Collect: company/name, Telegram id (primary), expected RPS, VPS spec, use case.
2. Check trial registry (Section 5).
3. Send link to **PUBLIC_OFFER** + pilot checkbox: "I accept the Public Offer for pilot license."
4. Issue `pilot` JWT (14 days, 5k RPS, rules-only ML/RTB).
5. Log: `deployment_id`, telegram, timestamp, offer version.

### 2.2 Paid conversion

1. Send [INVOICE.md](./INVOICE.md) template with SKU, USDT amount, **same `deployment_id`** as pilot.
2. Buyer pays USDT; buyer replies with tx hash.
3. Verify payment; issue paid SKU JWT with `--mark-converted`.
4. Acceptance = payment + prior offer acceptance (or explicit checkbox on invoice reply).

### 2.3 Direct paid (no pilot)

1. Buyer reads PUBLIC_OFFER on site.
2. Checkout: SKU, price, checkbox, payment.
3. Issue JWT within SLA on invoice (Pro/Scale 12h; others 24h per INVOICE.md).

### 2.4 Evidence to retain

| Field | Why |
| :--- | :--- |
| Offer version + URL | Proves terms shown before payment |
| Timestamp + IP / Telegram id | Acceptance audit |
| SKU + amount + tx hash | Price and performance |
| `deployment_id` | Renewal and support |
| HWID / fingerprint | Host binding disputes |

---

## 3. FAQ scripts (buyer questions)

### 3.1 "Do you store our data?"

**Answer:** No. ad-event-processor is **self-hosted**. Campaigns, clicks, conversions, and reports live on **your** servers (Postgres, Redis, ClickHouse on your VPS). We deliver binaries and an offline license file. We do not operate your production database or see your traffic in the normal product model.

**Do not say:** "We are GDPR compliant for your users" — buyer remains controller.

### 3.2 "Are you GDPR compliant?"

**Answer:** For the standard product, we do not process your end-user personal data — you do, on your infrastructure. You need your own privacy policy, lawful basis, and DPA with your host/CDN. We process only **your** contact and billing data to sell and support the license. A GDPR Article 28 DPA with us is needed only if you buy optional managed services where we access your systems.

### 3.3 "Is this SaaS / cloud?"

**Answer:** No. You run Docker compose (or release images) on your VPS. One install command: `bash scripts/install/ad-event-processor-install.sh up`. We do not host tenants.

### 3.4 "What happens if we don't pay renewal?"

**Answer:** License JWT expires. You get **7 days grace** (default), then ingest is **blocked** (`EXPIRED`). Admin may show renewal banner. Your data stays on your disks — we do not delete it. Renew by paying and applying a new JWT with the **same deployment_id**.

### 3.5 "Can we move to another server?"

**Answer:** Depends on SKU bind mode. `starter`/`pilot`: **hard** bind (one host). `pro`/`scale`/`network`: **multi** bind up to `max_activations`. Contact support **before** migration; we may re-issue JWT with new HWID. Do not promise unlimited moves.

### 3.6 "What RPS / campaigns limit?"

**Answer:** Quote **peak RPS** and **host count** from [SALES.md](./SALES.md). `max_active_campaigns: 0` in SKU means **no license cap** on campaigns — hardware and RPS are the practical limits.

### 3.7 "Does ML block fraud on every click?"

**Answer (honest):** No. Rule filters run on the hot path. **ML fraud boost** and **IVT detector** are **batch sidecars** on ClickHouse (Scale+). Do not promise real-time ML on every `/track`. See [MARKETING.md](./MARKETING.md) fraud section.

### 3.8 "OpenRTB included?"

**Answer:** From **Scale** SKU up. Starter/Pro are click URL + S2S `/track` only.

### 3.9 "Do you guarantee uptime / ROI?"

**Answer:** No contractual uptime on buyer infrastructure. Technical docs cite engineering targets, not warranties. ROI depends on buyer traffic and setup.

### 3.10 "Phone-home / telemetry?"

**Answer:** License verifies **offline** (Ed25519 JWT). Default install: `TELEMETRY_ENABLED=false`. No license server ping required.

### 3.11 "Admin UI included?"

**Answer (current tree):** HTTP API on `:8188` is supported. Full React admin may require a build that includes `web/dist`. Do not promise a feature-complete browser console unless the buyer receives that build. API and install scripts are the contract baseline.

### 3.12 "Refund if it doesn't work?"

**Answer (B2B):** Fees non-refundable after JWT delivery except if we fail to deliver a valid token within 5 business days after confirmed payment. **Pilot is free.** Troubleshooting is best-effort install support in first paid month.

### 3.13 "Ukraine / EU law?"

**Answer:** Offer governed by **Ukraine law**. EU buyers doing business with EU data subjects must implement their own GDPR program on their deployment.

### 3.14 "Is this legal? What about Article 361 KKU?"

**Answer (dual-use framing):**

> ad-event-processor is **dual-use technology** with a clear lawful purpose: click tracking, postbacks, campaign budgets, and antifraud on **your** infrastructure. Like a knife or a database, it does not create criminal intent by itself.
>
> We **do not store** your campaigns or traffic on our side. We **do not ship** features whose purpose is breaking into third-party systems (no exploit framework, no credential cracker, no botnet C2).
>
> **CC Article 361** punishes the person who **unlawfully interferes** with someone else's systems. That is **your responsibility as the operator** who configures sources, integrations, and legal bases — not the liability of a vendor who delivered lawful server software for advertising.
>
> The public offer allocates criminal and civil liability accordingly (Sections 3.6, 9.1, 11.5).

**If prospect wants exploit/hacking tooling or clearly illegal use:** decline sale; note refusal in CRM.

### 3.15 "Will you be liable if we get investigated?"

**Answer:** We are not a co-perpetrator of acts you perform on your VPS. You indemnify us if third parties try to drag the vendor into your case (Offer Section 12). We can revoke the license if we receive credible evidence of unlawful use — that is a commercial safeguard, not an admission of guilt.

---

## 4. Tier selection (quick guide)

| Buyer | Recommend | Why |
| :--- | :--- | :--- |
| Solo affiliate, rules-only | `starter` | $129, 10k RPS, 1 host |
| Media buyer + IVT reports | `pro` | IVT on ClickHouse |
| Network + OpenRTB + ML | `scale` | RTB, ML boost, intel feeds |
| Multi-region | `network` | `multi_region`, 10 hosts |
| XDP edge + platform API sync | `enterprise` | Custom quote $2500+ |

Full matrix: [SALES.md](./SALES.md) and [sku.yaml](./sku.yaml).

**Upsell lines:**
- Pro -> Scale: "OpenRTB and ML boost need Scale."
- Scale -> Network: "Second region needs Network SKU."

---

## 5. Trial registry (repeat pilot defense)

Env: `VENDOR_TRIAL_REGISTRY=deploy/vendor/trial_registry.json`

**Reject pilot when:**
- Same `telegram` had prior pilot `active` or `expired`
- Same `hwid` / HWID v2 had prior pilot
- Same USDT wallet funded prior pilot (paid conversion tracking)

**Force re-issue (internal only):** `VENDOR_TRIAL_FORCE=1` + `--force --force-reason "<ticket>"`

**CLI:**
```bash
go run ./cmd/trial-registry list-pending
go run ./cmd/license-issue --approve-pending <pending_id>
```

---

## 6. License issuance runbook

### 6.1 Prerequisites

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key
# Never commit private key or issued JWTs to git
```

### 6.2 Collect from buyer

| Field | Required when |
| :--- | :--- |
| Customer legal name | Always |
| `deployment_id` (UUID) | Always; **reuse** on renewal/upgrade |
| SKU code | Always |
| `--hwid-v2` hash | `hard` / `pilot` bind; recommended always |
| `--telegram-id` | Pilot |
| USDT tx hash | Paid conversion |

**HWID collection (buyer runs on VPS):**
```bash
bash scripts/lab/hwid_collect.sh
# or GET /api/v1/license/status after first boot -> hwid_v2
```

### 6.3 Issue pilot

```bash
go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<64-char-hex>" \
  --telegram-id "<telegram_user_id>" \
  --out /tmp/acme-pilot.jwt
```

### 6.4 Issue paid (after payment)

```bash
go run ./cmd/license-issue \
  --sku pro \
  --customer "Acme Media" \
  --deployment-id "<same-uuid-as-pilot>" \
  --hwid-v2 "<hash>" \
  --mark-converted \
  --out /tmp/acme-pro.jwt
```

### 6.5 Renewal (monthly)

Same command with new `--valid-days` from SKU (default 30); **same `deployment_id`**.

### 6.6 Delivery to buyer

Send JWT via encrypted channel. Instructions:

1. Admin: Settings -> License -> paste -> Apply.
2. CLI: `bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'`
3. Verify: `GET /api/v1/license/status` -> `ACTIVE`, correct `plan_code`, `days_to_expiry`.

**SLA (from INVOICE.md):** Pro/Scale 12h; Starter/Network/Enterprise 24h after tx confirmation.

---

## 7. Deployment consultation checklist

Walk buyer through **before** promising go-live date.

### 7.1 Hardware

| Profile | RAM hint | Notes |
| :--- | :--- | :--- |
| `ingest-only` | 6-8 GB | No ClickHouse |
| `full` / `single-vps` | 16+ GB | Analytics + admin |
| `analytics-ml` | +2-4 GB | fraud-scorer, ivt-detector |

Preflight: `bash scripts/install/preflight.sh`

### 7.2 OS and edge

- Linux VPS with Docker Compose v2.
- **eBPF XDP** (Enterprise): kernel >= 6.1, BTF, `CAP_BPF` — see [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md).
- Do not promise XDP on laptop or kernel without BTF.

### 7.3 Install steps (buyer)

```bash
# Release tarball or git clone on buyer VPS
cp .env.example .env
# Edit secrets, REDIS_ADDRS, domains
bash scripts/install/ad-event-processor-install.sh --accept-eula up
```

Dev/bootstrap (not production): `bash scripts/install/appliance_bootstrap.sh --profile full`

### 7.4 Post-install

| Step | Command / endpoint |
| :--- | :--- |
| Apply license | `license-apply` or Admin API |
| Platform bootstrap | Installer runs `POST /api/v1/settings/platform/bootstrap` |
| Doctor | `ad-event-processor-install.sh doctor` |
| License status | `GET /api/v1/license/status` |
| Smoke | `bash scripts/dev/stack/preflight.sh` (after stack up) |

### 7.5 GeoIP

Buyer needs MaxMind GeoLite2 or license. Bootstrap can fetch if `MAXMIND_LICENSE_KEY` in `.env`.

### 7.6 What we do vs buyer ops

| We (vendor) | Buyer |
| :--- | :--- |
| JWT, tier features | VPS, firewall, backups |
| Install guidance (1st month) | TLS certs, DNS |
| Defect triage on reference profile | CDN config, traffic volume |
| Re-issue JWT on renewal | Postgres/Redis/CH capacity |

---

## 8. Organisational workflows

### 8.1 Roles

| Role | Owns |
| :--- | :--- |
| Sales | Qualification, offer acceptance, invoice |
| Vendor ops | `license-issue`, trial registry |
| Support | Install tickets, `doctor` output |
| Legal | PUBLIC_OFFER placeholders, B2C notices |

### 8.2 Escalation

| Issue | Escalate to |
| :--- | :--- |
| HWID mismatch / bind deny | Ops + re-issue with verified hash |
| Repeat trial abuse | Ops; deny or force with reason |
| Feature gate dispute | Sales + SALES.md matrix |
| Security incident on buyer VPS | Buyer; we have no access unless contracted |
| Consumer complaint (UA) | Legal within 48h |

### 8.3 Enterprise exceptions

Written addendum for: custom RPS, SLA, on-site install, DPA for managed access, multi-year prepay.

---

## 9. Forbidden sales claims

From [MARKETING.md](./MARKETING.md) and engineering truth:

| Do not claim | Truth |
| :--- | :--- |
| "We host your campaigns" | Self-hosted only |
| "AI blocks all bots live" | ML is batch; rules on hot path |
| "XDP stops residential proxies" | L4 flood + known IPs; rotating proxies evade |
| "202 = written to Postgres/CH" | 202 = hot path accepted; async sink |
| "Unlimited everything" | RPS + hosts + features are SKU-gated |
| "Full admin UI out of the box" | API yes; UI depends on build |
| "GDPR compliant hosting" | Buyer is controller of traffic data |
| "Use for account takeover / scraping protected sites" | CC Art. 361 criminal offense; license forbids |
| "We take responsibility if buyer misuses antifraud" | Buyer indemnifies vendor (Offer Section 12) |

---

## 10. Checkout / website implementation

1. Footer link: `PUBLIC_OFFER.md` (rendered HTML/PDF).
2. Checkbox mandatory before Pay / Request pilot.
3. Show SKU, USDT price, period, RPS, hosts.
4. B2C: statutory withdrawal notice for digital content (legal draft).
5. Log acceptance event server-side.
6. Invoice email references offer version date.

---

## 11. Related commands reference

```bash
# Issue license
go run ./cmd/license-issue --sku <code> --customer "..." --deployment-id <uuid> --out token.jwt

# Trial registry
go run ./cmd/trial-registry list-pending

# License verify (engineering QA)
make license-verify

# Install on buyer machine
bash scripts/install/ad-event-processor-install.sh up
bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'
bash scripts/install/ad-event-processor-install.sh doctor
```

---

## 12. Document index

| File | Use |
| :--- | :--- |
| [PUBLIC_OFFER.md](./PUBLIC_OFFER.md) | Customer-facing offer (EN + UK) |
| [SALES.md](./SALES.md) | SKU math and enforcement map |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [KEYS.md](./KEYS.md) | Ed25519 keys, HWID v2 params |
| [MARKETING.md](./MARKETING.md) | Honest feature list for prospects |
| [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md) | XDP / multi-region |
| [VENDOR.md](./VENDOR.md) | Vendor tree index |

**Revision log:** maintain in git commit messages; update `PUBLIC_OFFER.md` version when terms change.
