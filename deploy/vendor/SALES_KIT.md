# BidShard commercial sales kit (internal)

Vendor-only. Not shipped in customer tarballs. Canonical SKU limits: [sku.yaml](./sku.yaml). Issue JWTs with `go run ./cmd/license-issue --sku <code> …`.

## Positioning

- **Solo / Starter** — on-prem tracker + Cost Sync; HTTP redirect + `/track` only (OpenRTB blocked). Noise shield vs Klixsor / Keitaro.
- **Pro / Scale** — programmatic stack (OpenRTB live); host cap via `max_activations`.
- **Network / Enterprise** — multi-region, ML cold path; **Enterprise** unlocks eBPF/XDP edge.

**Enforcement model:** limit by **active campaigns** + **host fingerprint** (`max_activations`), not buyer seats (shared login bypass).

## Tier ladder (USDT / month, on-prem, monthly JWT)

| Tier | SKU | USDT/mo | Setup | Hosts | Campaigns | Clicks/day | RPS |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| **Solo** | `solo` | **$69** | $100 | 1 | 10 | 100k | 2k |
| **Starter** | `starter` | **$99** | $150 | 1 | 25 | 500k | 5k |
| **Pro** | `pro` | **$199** | $150 | **1** | **50** | 2M | 15k |
| **Scale** | `scale` | **$399** | $200 | **3** | 150 | 5M | 50k |
| **Network** | `network` | **$799** | $250 | 10 | 500 | 10M | 100k |
| **Enterprise** | `enterprise` | **$1,999** | custom | 99 | unlimited | custom | custom |

Runtime gates: `SanitizeFeaturesForSKU`, `OpenRTBAllowed`, `EbpfEdgeAllowed`, `CheckHostActivation`, campaign cap on create, `LicenseRPSFilter` + daily quota in `EntitlementsFilter`.

## Competitor anchors (Aug 2026, public)

| Product | License/mo | Notes |
| :--- | ---: | :--- |
| Klixsor | $49 | ~10 campaigns; TrustMRR ~3 subs |
| Keitaro | $25–70 | + VPS $30–80 |
| Binom v2 self-hosted | $149 | Unlimited events, ClickHouse |
| RedTrack / ClickFlare | $89–199 | Cloud, CAPI at entry |
| Voluum | $199–999+ | Per-event cloud |

## Total cost to buyer (license + VPS — always quote both)

| Stack | License | VPS (typical) | **Total/mo** |
| :--- | ---: | ---: | ---: |
| Klixsor Solo analog | $69 | $7–40 | **$76–109** |
| BidShard Starter | $99 | $40–60 | **$139–159** |
| Binom v2 | $149 | $40–80 | **$189–229** |
| BidShard Pro | $199 | $40–80 | **$239–279** |
| BidShard Scale | $399 | $80–120 | **$479–519** |

## Feature matrix (buyer-facing)

| Capability | Solo | Starter | Pro | Scale | Network+ |
| :--- | :---: | :---: | :---: | :---: | :---: |
| Cost Sync UI | — | ✓ | ✓ | ✓ | ✓ |
| CAPI (Meta/Google/TikTok) | — | Meta only | ✓ | ✓ | ✓ |
| RTB `/openrtb/bid` live | blocked | blocked | live | live | live |
| Margin Guard | ✓ | ✓ | ✓ | ✓ | ✓ |
| Offline JWT + hard bind | ✓ | ✓ | ✓ | ✓ | ✓ |
| Telegram vendor support | best-effort | 48h | 24h | 12h | SLA chat |
| `ivt_ml` / fraud-scorer | — | — | optional | ✓ | ✓ |
| eBPF XDP edge | blocked | blocked | blocked | blocked | Enterprise only |

## Pilot → paid (GTM)

| Phase | Price | Duration | JWT SKU |
| :--- | :--- | :--- | :--- |
| **Pilot** | $0 | 30 days | `pilot` (35-day JWT, hard bind) |
| **Lock-in** | $99 | months 2–7 | `starter` limits — **not** `pilot` |
| **Standard** | tier table | month 8+ | Full tier price |

Do **not** grant pilot limits ($50k RPS / 5M day) at $99 — issue `starter` or `pro` JWT with matching `limits` in [sku.yaml](./sku.yaml).

## Revenue targets (solo vendor, realistic)

| Scenario | Clients | Avg USDT/mo | MRR |
| :--- | ---: | ---: | ---: |
| Floor | 3 | $99 | **$297** |
| Base (Sep–Dec 2026) | 8 | $149 | **$1,192** |
| Stretch | 15 | $199 | **$2,985** |

**$1k–1.5k MRR by Dec 2026** = base case. **$5k+ MRR** needs 3–5 public case studies (p99 + True ROI).

## SLA (business)

| Item | SLA |
| :--- | :--- |
| USDT confirmed → renewal JWT | **24 h** (Pro/Scale **12 h**) |
| Setup fee | One install assist (≤ 2 h Telegram); redeploys self-serve via [QUICKSTART](../../docs/QUICKSTART.md) |
| Tier upgrade | New JWT only — no reinstall ([QUICKSTART § License tier upgrade](../../docs/QUICKSTART.md#license-tier-upgrade-no-reinstall)) |

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

Send [USDT invoice](./USDT_INVOICE_TEMPLATE.md) first for new paid tiers; after on-chain confirm, email JWT line + short renewal instructions ([PILOT_LICENSE.md](../../docs/PILOT_LICENSE.md)).
