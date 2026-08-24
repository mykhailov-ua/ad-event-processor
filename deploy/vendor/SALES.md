# BidShard Commercial Sales Kit (Internal)

Vendor-only. Confidential. Not shipped in customer packages. Runtime limits are configured in [sku.yaml](./sku.yaml). Issue JWTs using `go run ./cmd/license-issue --sku <code> ...`.

---

## Market Positioning

BidShard is positioned as the premium, enterprise-grade on-premise alternative to both legacy self-hosted trackers and expensive SaaS traffic platforms.

- **Starter Tier** - Perfect for individual high-volume buyers, solo media buyers, and affiliate teams. Focuses on rapid HTTP redirection, `/track` ingestion, and Cost Sync (no OpenRTB engine). Positioned to crush entry-level competitors like Keitaro and Binom.
- **Pro / Scale Tiers** - Engineered for growing media buying teams and programmatic desks. Unlocks the OpenRTB bidding engine for native, low-latency DSP/SSP buying, with higher host counts.
- **Network / Enterprise Tiers** - Designed for large networks, agency desks, and custom high-volume operations. Unlocks multi-region deployment, machine-learning-driven cold-path analysis (AI Antifraud), eBPF/XDP edge security, and customizable service SLAs.

### Flat-Rate Performance Billing Model
We charge strictly for **hosts (nodes)** (`max_activations`) and **peak Request-Per-Second (RPS)** caps. 

We offer **unlimited campaigns** and **no monthly event/click volume limits**. Self-hosted buyers own their servers and bandwidth; throttling them with SaaS-style monthly event limits is counterproductive. Peak RPS is the only honest metric that aligns with actual hardware capability.

---

## Value Proposition: Self-Hosted vs. SaaS

| SaaS-Era Limitation | Why It Fails in Self-Hosted (2026) | BidShard Advantage |
| :--- | :--- | :--- |
| **Active Campaign Caps** | Facebook/TikTok testing easily spawns 30–50 campaigns in a few days. Caps choke buyer activity before scale is reached. | **Unlimited Campaigns.** No artificial gates on the customer’s database or disk. |
| **Monthly Event Quotas** | Charging per million events on customer's own hardware feels like double-billing (VPS + License). | **Unlimited Events.** Free scale for the buyer; bandwidth is only gated by peak hardware capability. |
| **Daily Click Quotas** | Punishes low-RPS testing and standard weekend spikes; forces buyers to guess their monthly needs. | **Peak RPS Billing.** Predictable and fair. Licensed RPS handles standard spikes seamlessly. |
| **Active Database Row Limits** | Keeps trackers complex and penalizes archiving; 100 idle campaigns add zero CPU load. | **Unlimited Scaling.** Local ClickHouse handles billions of events without degradation. |

*Sales Instruction:* Quote strictly by **RPS** and **active host count**. Never refer to "X events per month" or "X campaigns included" when discussing BidShard.

---

## Product Pricing Evolution

Why the legacy pricing tier model failed under older structures, and how we optimized it:

| Challenge | Old Structure | business Impact | BidShard New Tier Alignment |
| :--- | :--- | :--- | :--- |
| **Solo vs. Entry Overlap** | $69 (10 campaigns) + $99 (25 campaigns) + $150 setup fee | Worse value than Keitaro and Binom. Setup fees killed trial traffic conversion. | **Removed campaign caps.** Removed setup fees. Starter tier is $129/mo with *unlimited* campaigns on 1 host. |
| **Pro Underpriced for RTB** | $199 for 15k RPS with RTB | Programmatic buyers pay $1k+/mo for similar in-house DSP/SSP ingestion. | **Pro Tier is $329/mo.** Standardized pricing vs. actual market value of real-time programmatic bidding. |
| **Scale Underpriced** | $399 for 3 hosts / 50k RPS | Large agencies treat $399 as "hobbyist" pricing, signaling low reliability. | **Scale Tier is $649/mo.** Aligns with large media-buying teams running massive concurrent campaigns. |
| **Enterprise Underpriced** | $1,999 (eBPF + ML) | Enterprise buyers infer security risks or poor SLA below a $2.5k floor. | **Enterprise is $2,500+/mo.** Captures true corporate budgets and covers dedicated SLA and customization. |
| **Setup Fee Barrier** | $100–$250 upfront | Creates a mental barrier for buyers testing an unfamiliar brand. | **$0 Setup Fee.** Replaced with *"Installation support included with your first paid month"*. |
| **Trial Cycle Too Long** | 30-day free trials | Users extract fully working funnels, run quick campaigns, and churn. | **10-day Pilot.** High-intensity test to verify latency, integrations, and CAPI, preventing free-riding. |

---

## Standard Tier Pricing (USDT / Month, Self-Hosted)

All licenses are paid in USDT and issued as monthly, cryptographic offline JWT files.

| Tier | SKU | USDT/mo | Setup Fee | Nodes (Hosts) | Peak RPS | Campaigns | Event Volume |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **Starter** | `starter` | **$129** | **$0** | 1 | 10k | Unlimited | Unlimited |
| **Pro** | `pro` | **$329** | **$0** | 1 | 25k | Unlimited | Unlimited |
| **Scale** | `scale` | **$649** | **$0** | 3 | 75k | Unlimited | Unlimited |
| **Network** | `network` | **$1,199** | Included | 10 | 150k | Unlimited | Unlimited |
| **Enterprise** | `enterprise` | **$2,500+** | Custom | 99 | Custom | Unlimited | Unlimited |
| **Pilot (Trial)** | `pilot` | **$0** | **$0** | 1 | 5k | Unlimited | Unlimited |

*Negotiation Range:* Starter **$119–$149**, Pro **$299–$349**, Scale **$599–$699**. Default to the standard table pricing above.

---

## Infrastructure Sizing Guides (Starter)

ClickHouse is **not** a license-gated feature—it is a deployment profile choice. Campaign definitions, state, and monetary settlements are stored in **PostgreSQL**. The tracker hot-path never blocks on ClickHouse.

| Profile | Active Services | Ideal Customer Use Case | Typical VPS Cost |
| :--- | :--- | :--- | :--- |
| **Ingest-Only** | Tracker, Processor, Control API, PG, Redis x4 (No ClickHouse) | Solo buyers, direct redirects, Facebook/Meta CAPI integration, Postgres Cost Sync. | **$40–$60 / mo** (6–8 GB RAM) |
| **Single-VPS** | All above + ClickHouse analytical database | Teams requiring true ROI reports, hourly placement breakdowns, and smart metrics. | **$60–$80 / mo** (16+ GB RAM) |

*Sales Guidance:* Recommend the **Ingest-Only** profile to solo buyers to save on hosting costs. Upsell the **Single-VPS** profile when they ask for rich multidimensional reporting.

---

## Market Competitor Anchors (2026)

| Competitor | Pricing / Month | Deployment | BidShard Advantage vs. Competitor |
| :--- | :--- | :--- | :--- |
| **Keitaro** | €40–€70 (~$45–$75) | Self-hosted | BidShard offers enterprise-grade latency (p99 < 80ms) and highly advanced network-layer antifraud. |
| **Binom v2** | $149 | Self-hosted | BidShard provides out-of-the-box Meta/TikTok Conversions API (CAPI) and programmatic OpenRTB bidding. |
| **Voluum / RedTrack** | $199–$999+ | Cloud SaaS | BidShard eliminates click-overage fees, offers true data privacy, and prevents spy-tool scraping. |

---

## Feature Comparison Matrix

| Feature | Starter | Pro | Scale | Network+ |
| :--- | :---: | :---: | :---: | :---: |
| **Cost Sync UI** | Yes | Yes | Yes | Yes |
| **Conversions API (CAPI)** | Meta Only | Yes | Yes | Yes |
| **Programmatic OpenRTB** | No | Yes | Yes | Yes |
| **Margin Guard** | Yes | Yes | Yes | Yes |
| **Offline Cryptographic JWT** | Yes | Yes | Yes | Yes |
| **Telegram Tech Support** | 48h SLA | 24h SLA | 12h SLA | Dedicated VIP Chat |
| **AI Antifraud (ML Scorer)** | No | Optional | Yes | Yes |
| **eBPF/XDP Kernel Filter** | No | No | No | Enterprise Only |
| **Residential IP Intelligence** | No | No | Yes | Yes |
| **Peak Ingestion RPS** | 10k | 25k | 75k | 150k+ |

---

## Pilot-to-Paid Sales Pipeline

1. **Pilot Phase ($0 / 10 Days):** Issued using SKU `pilot` (5k RPS limit, OpenRTB disabled, hard hardware bind). Used exclusively to verify server latency, tracking setup, and Conversions API integration.
2. **Paid Conversion:** After 10 days, the pilot license expires. To renew, the client pays the monthly USDT fee, and we issue a regular `starter`/`pro`/`scale` monthly license with matching limits in `sku.yaml`. We do not extend pilot periods beyond a 7-day grace window, and only for written, high-intent enterprise use cases.

---

## Support & Operations SLAs

- **USDT Confirmation to Renewal JWT:** **24 hours** max (Starter/Network/Enterprise), **12 hours** max (Pro/Scale).
- **Onboarding Assistance:** Included with first paid month. Up to 2 hours of direct Telegram setup/install assistance. Redeployments are fully self-service via `docs/DEVELOPMENT.md`.
- **Tier Upgrades:** Handled by issuing a new JWT instantly. No server reinstallation or downtime is required.
