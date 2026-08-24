# ad-event-processor commercial sales kit (internal)

Vendor-only. Not shipped in customer tarballs. Runtime limits: [sku.yaml](./sku.yaml). Issue JWTs with `go run ./cmd/license-issue --sku <code> …`.

## Positioning

- **Starter** — on-prem tracker + Cost Sync; HTTP redirect + `/track` only (OpenRTB blocked). Entry tier vs Keitaro / Klixsor.
- **Pro / Scale** — programmatic stack (OpenRTB live); host cap via `max_activations`.
- **Network / Enterprise** — multi-region, ML cold path; **Enterprise** unlocks eBPF/XDP edge + custom SLA.

**Enforcement model:** **hosts** (`max_activations`) + **peak RPS** + **feature gates** (OpenRTB, ML, eBPF). **Unlimited campaigns** and **no event-volume caps** — self-hosted buyers own the disk and VPS; throughput is gated by instantaneous RPS and host count, not SaaS-style monthly click accounting. No buyer-seat metering (shared login bypass).

## Why no campaign or event-volume caps (self-hosted 2026)

| SaaS-era metric | Why it fails on-prem |
| :--- | :--- |
| Active campaign count | FB/TT tests spawn 30–50 campaigns in days; cap hits before traffic matters |
| Events / month | Voluum-style billing on **your** hardware; buyer already paid VPS + license |
| Daily click quota | Same as monthly — punishes low-RPS tests and weekend spikes; RPS cap is the honest signal |
| DB row limits | 100 idle campaigns ≈ zero tracker load; one pop campaign can burn 50k RPS |
| Binom v2 anchor | "Unlimited campaigns on your hardware" is table stakes for self-hosted |

Quote **RPS** and **hosts** only. Never "X events/month" or "X campaigns included".

## Why the old ladder failed (Aug 2026 review)

| Problem | Old state | Impact |
| :--- | :--- | :--- |
| Solo + Starter overlap | $69 / 10 camp + $99 / 25 camp + $100–150 setup | Worse than Keitaro (€40 entry) and Binom ($149 unlimited); setup fee kills cold traffic |
| Campaign caps | 10–500 per tier | Artificial gate on buyer's own disk; instant churn signal |
| Pro underpriced for RTB | $199 / 15k RPS | In-house DSP/SSP pays $800–1,500/mo for comparable ingress |
| Scale underpriced | $399 / 3 hosts / 50k RPS | Large shops treat $400 as noise — signals low reliability |
| Enterprise too cheap | $1,999 with eBPF + ML | Enterprise buyers infer risk below ~$2.5k |
| Pilot too long | 30 days | Enough to extract a bundle and churn before first invoice |
| Setup fee as barrier | $100–250 upfront | Blocks closes for unknown brand |

## Tier ladder (USDT / month, on-prem, monthly JWT)

| Tier | SKU | USDT/mo | Setup | Hosts | Peak RPS | Campaigns | Event volume |
| :--- | :--- | ---: | :---: | ---: | ---: | :---: | :---: |
| **Starter** | `starter` | **$129** | **$0** | 1 | 10k | unlimited | unlimited |
| **Pro** | `pro` | **$329** | **$0** | 1 | 25k | unlimited | unlimited |
| **Scale** | `scale` | **$649** | **$0** | 3 | 75k | unlimited | unlimited |
| **Network** | `network` | **$1,199** | incl. | 10 | 150k | unlimited | unlimited |
| **Enterprise** | `enterprise` | **$2,500+** | custom | 99 | custom | unlimited | unlimited |

Price bands for negotiation: Starter **$119–149**, Pro **$299–349**, Scale **$599–699**. Default quote = mid-band above.

**Setup / onboarding:** no separate setup line item. Copy: *«Install included with first paid month»*. First month invoice = license only. One Telegram install assist (≤ 2 h) on Starter+; redeploys self-serve via [DEVELOPMENT.md](../../docs/DEVELOPMENT.md).

Runtime gates: `SanitizeFeaturesForSKU`, `OpenRTBAllowed`, `EbpfEdgeAllowed`, `CheckHostActivation`, `LicenseRPSFilter` (licensed RPS + **10% burst** for ~45s before 429). Canonical limits: [sku.yaml](./sku.yaml) (`max_events_per_month: 0`, `max_requests_per_day: 0`, `max_active_campaigns: 0` = not enforced).

## Starter stack sizing (ClickHouse optional)

ClickHouse is **not** a license gate — it is a deploy profile choice. Settlement and campaign truth live in **Postgres**; hot path never blocks on CH.

| Profile | Services | When to quote |
| :--- | :--- | :--- |
| **ingest-only** (`stack.sh ingest-only`) | tracker, processor, control, PG, Redis ×4 — **no CH** | Solo buyer, redirect + `/track`, Meta CAPI, Cost Sync to PG. **Default Starter pitch.** |
| **single-vps** | above + ClickHouse | Buyer wants True ROI, placement hourly, Smart Alerts on CH metrics |

**RAM guide (1 host):** ingest-only ≈ **6–8 GB** (CX31). Add CH ≈ **+2 GB** → CX41 / 16 GB if they need analytics on-box.

Do not force Binom-style "CH included" on entry — quote **$40–60 VPS** without CH; upsell CH stack when they ask for reports beyond PG aggregates.

## Competitor anchors (Aug 2026, public)

| Product | License/mo | Notes |
| :--- | ---: | :--- |
| Keitaro | €40–70 (~$45–75) | + VPS $30–80; entry anchor for media buyers |
| Klixsor | $49 | ~10 campaigns (SaaS-style cap) |
| Binom v2 self-hosted | $149 | Unlimited campaigns, ClickHouse |
| RedTrack / ClickFlare | $89–199 | Cloud, CAPI at entry |
| Voluum | $199–999+ | Per-event cloud |
| In-house OpenRTB ingress | $800–1,500+ | Not a tracker — reference for Pro/Scale pricing |

## Total cost to buyer (license + VPS — always quote both)

| Stack | License | VPS (typical) | **Total/mo** |
| :--- | ---: | ---: | ---: |
| Keitaro + VPS | $45–75 | $30–80 | **$75–155** |
| ad-event-processor **Starter** (ingest-only) | $129 | $40–60 | **$169–189** |
| ad-event-processor **Starter** + CH reports | $129 | $60–80 | **$189–209** |
| Binom v2 | $149 | $40–80 | **$189–229** |
| ad-event-processor **Pro** (OpenRTB) | $329 | $40–80 | **$369–409** |
| ad-event-processor **Scale** (3 hosts) | $649 | $80–120 | **$729–769** |

Starter is priced above Keitaro on purpose: filters non-payers and funds real support. Pro/Scale are priced vs programmatic infra and host/RPS envelope, not campaign count.

## Feature matrix (buyer-facing)

| Capability | Starter | Pro | Scale | Network+ |
| :--- | :---: | :---: | :---: | :---: |
| Cost Sync UI | ✓ | ✓ | ✓ | ✓ |
| CAPI (Meta/Google/TikTok) | Meta only | ✓ | ✓ | ✓ |
| RTB `/openrtb/bid` live | blocked | live | live | live |
| Margin Guard | ✓ | ✓ | ✓ | ✓ |
| Offline JWT + hard bind | ✓ | ✓ | ✓ | ✓ |
| Telegram vendor support | 48h | 24h | 12h | SLA chat |
| `ivt_ml` / fraud-scorer | — | optional | ✓ | ✓ |
| eBPF XDP edge | blocked | blocked | blocked | Enterprise only |
| External residential intel (cold) | blocked | blocked | ✓ (JWT) | ✓ (JWT) |
| Quoted limits (sales) | 10k RPS, 1 host | 25k RPS, 1 host | 75k RPS, 3 hosts | custom |

## Antifraud capability matrix (shipped vs backlog)

Sales-facing map of **what runs today** vs SKU gates. Backlog IDs = repo [BACKLOG.md](../../BACKLOG.md) (do not promise unshipped rows).

| Capability | Shipped | Backlog | SKU / config gate |
| :--- | :---: | :--- | :--- |
| IPv6 edge LPM + `/click` rotation velocity | ✓ | P2-1 – P2-3 | Tracker env `IPV6_ROTATION_*`; edge XDP IPv6 maps |
| Residential proxy hot ring | ✓ | P3-1 – P3-6 | `RESIDENTIAL_PROXY_HOT_ENABLED` |
| External residential intel (cold enricher) | ✓ | P3-* | `external_residential_intel` JWT; Scale+ |
| Safe page + L2 attestation + headless signals | ✓ | P4-1 – P4-4 | Campaign `safe_page_*`, `attestation_*` |
| Social in-app WebView (FB/TikTok/IG) | ✓ | P5-1 – P5-3 | Preset `social_in_app`; TLS allowlist feed |
| Local quanta full-skip (0 sync Lua) | ✓ | P6-1 | `LOCAL_QUOTA_MODE=live` + ledger credit |
| Lua placement/RPD precheck in Go | ✓ | P6-2 | UnifiedFilter + EntitlementsFilter |
| XDP blocklist incremental sync (500k+) | ✓ | P6-4 | `ebpf_xdp_edge` Enterprise only |
| Budget/fcap sub-shards (hot campaigns) | ✓ | P6-5 | `BehaviorHighVolumeDebit` on campaign |
| `ivt_ml_detector` batch rules + ML boost | ✓ | — | Scale+ JWT (`ivt_ml_detector`, `ml_fraud_boost`) |
| eBPF NIC drop | ✓ | P6-4 | Enterprise JWT (`ebpf_xdp_edge`) |

**Not sold as unlimited QPS:** license enforces `max_rps` per tier; 100k+ QPS requires Scale/Network sizing + load proof (see BACKLOG Phase 6 verify), not mock benches.

## Pilot → paid (GTM)

| Phase | Price | Duration | JWT SKU |
| :--- | :--- | :--- | :--- |
| **Pilot** | $0 | **10 days** | `pilot` (`valid_days` = 10, 5k RPS, OpenRTB off, hard bind) |
| **Conversion** | tier table | month 1+ | `starter` / `pro` / … — **not** `pilot` limits |

Pilot goal: latency, stability, Cost Sync / CAPI smoke — not a free month of media buying. At day 10: convert or revoke; no automatic 30-day extension.

Do **not** grant pilot RPS/day quota on a paid Starter invoice — issue JWT with matching `limits` in [sku.yaml](./sku.yaml).

## Revenue targets (bootstrap vendor, realistic)

| Scenario | Clients | Avg USDT/mo | MRR |
| :--- | ---: | ---: | ---: |
| Floor | 3 | $129 | **$387** |
| Base (Sep–Dec 2026) | 8 | $200 | **$1,600** |
| Stretch | 15 | $280 | **$4,200** |

Higher ARPU assumes fewer tire-kickers (no setup fee) and 2–3 Pro/Scale programmatic deals. **$1.5k–2k MRR by Dec 2026** = base case. **$5k+ MRR** needs 3–5 public case studies (p99 + True ROI) and at least one Scale/Network logo.

## SLA (business)

| Item | SLA |
| :--- | :--- |
| USDT confirmed → renewal JWT | **24 h** (Pro/Scale **12 h**) |
| Onboarding | Included with first month; ≤ 2 h Telegram install assist |
| Tier upgrade | New JWT only — no reinstall (`.cursor/rules/licensing.mdc` — Customer apply) |
| Pilot extension | Case-by-case +7 days max; requires written use-case |

## Issue flow (vendor)

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key

go run ./cmd/license-issue \
  --sku pro \
  --customer "Acme Media" \
  --deployment-id "<uuid-from-customer-settings>" \
  --fingerprint "<host-fingerprint>" \
  --out /tmp/acme-pro.jwt
```

Send [USDT invoice](./USDT_INVOICE_TEMPLATE.md) first for new paid tiers (license line only — no setup fee row). After on-chain confirm, email JWT + renewal instructions ([licensing.mdc](../../.cursor/rules/licensing.mdc)).

Антифрод (питч и полный список фич): [ANTIFRAUD.md](./ANTIFRAUD.md).
