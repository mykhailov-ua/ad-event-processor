# Product overview (buyer-facing)

Honest summary of what **ad-event-processor** does today on a self-hosted install. This is not a managed SaaS. The buyer runs the stack on their own servers; the license is an offline JWT with RPS and host limits.

**Admin UI:** the React SPA (`web/`) is not shipped in this tree. Operators use the HTTP API (`/api/v1/*` on port 8188) and a minimal static boot stub until the UI is rebuilt. Do not promise a full browser console unless the buyer receives a build that includes `web/dist`.

---

## Who it is for

| Buyer | Fit |
| :--- | :--- |
| Solo affiliate / media buyer | Click tracking, postbacks, basic fraud rules, cost import, self-serve billing |
| Small buying team | Same + IVT rules on ClickHouse (Pro tier), team RBAC, reports |
| Network / high volume | OpenRTB exchange path, ML score boosts, intel feeds, multi-host (Scale+) |
| Multi-region operator | Regional proxy and slot migration (Network / Enterprise) |

---

## Core traffic (what buyers actually buy)

| Capability | What the buyer gets | Honest limits |
| :--- | :--- | :--- |
| Click tracking | `GET /click` — redirect with macros (`{campaign_id}`, `{click_id}`, `{sub1}`–`{sub30}`, UTMs) | Buyer wires traffic sources to the click URL |
| Server postbacks | `POST /track` — conversions and events (JSON, protobuf, or OpenRTB 3 ingress per campaign) | HTTP 202 means accepted on the hot path, not guaranteed Postgres/ClickHouse write yet |
| Browser pixel | `track.js` snippet — POST `/track` from landers with CORS | Requires buyer to embed snippet on landers |
| Telegram Mini App | `/tg/click`, `/tg/impression` | Telegram-specific flows only |
| OpenRTB exchange | `POST /openrtb/bid` — in-process auction, deals, floors | **Scale tier and up**; OpenRTB 3.0 and multi-imp >1 not implemented |
| Traffic templates | 82 bundled click schemas + 77 affiliate YAML templates | Custom schemas via admin API |

---

## Campaign and workflow

| Capability | Buyer value |
| :--- | :--- |
| Campaign CRUD | Budgets, schedules, geo, segments, status, pacing |
| Flows and landers | Path builder, hosted lander routes, offers |
| Brands and creatives | Brand assignment, creative library |
| Onboarding wizard | Template-based first campaign (`meta_social_funnel`, `popunder_propeller`, etc.) |
| Migration import | Preview/import from Keitaro, Binom, native JSON — validate before commit |
| Experiments / cohorts | A/B style cohorts on campaigns |
| Self-serve API keys | Advertisers manage campaigns via `/api/v1/selfserve/*` |

---

## Fraud and traffic quality

Sell as **layered defense**, not “AI blocks all bots.”

| Layer | Buyer-visible effect | Caveats |
| :--- | :--- | :--- |
| Edge (nginx) | Rate limits, circuit breaker, IP blacklist cache | Standard L7 perimeter |
| Go filters on tracker | Datacenter IP, low time-to-click, TLS fingerprint, device mismatch, proxy heuristics, attestation | Behind CDN without edge TCP/TLS sync, several signals fail-open or must be disabled |
| Silent reject | Decoy 202/302 instead of hard 403 when enabled per campaign | Analytics use `silent_reject_event`, not legacy “ghost” naming |
| Shadow mode | Event logged to analytics without budget debit when fraud signals fire | Buyer sees waste in reports, not blocked at edge |
| IP blacklist | Redis `blacklist:fraud`; optional XDP drop at NIC | **Enterprise** for eBPF XDP |
| IVT detector (Pro+) | Batch rules on ClickHouse — CTR spikes, bot intervals, tunnel RTT | Requires ClickHouse + `analytics-ml` profile; not real-time on every click |
| ML fraud boost (Scale+) | Batch scores adjust campaign thresholds via Redis | Runs in `fraud-scorer` sidecar, **not** on every `/track` request |
| Conversion smart reject | Processor rejects bad conversions (no click, low TTC, duplicate, IP drift) | Cold path; may defer payout until ClickHouse click row exists |
| Fraud evidence pack | Signed JSON export for CPA disputes (Scale+) | No raw IP in bundle; needs configured signing secret |
| Presets | `conservative`, `balanced`, `aggressive`, `enhanced_defense`, `social_in_app` | Operator tunes per campaign |

Do **not** claim: residential proxy detection on every request, guaranteed ML on hot path, or XDP stopping rotating proxies.

---

## Integrations and ROI

| Capability | Buyer value |
| :--- | :--- |
| Cost Sync | Daily spend import from **25** ad networks — true ROI vs revenue |
| Outbound CAPI | Meta, Google, TikTok conversion APIs + generic webhook |
| Postbacks | Per-campaign outbound postback configuration |
| Supply metadata | `sellers.json`, `ads.txt` export |
| Platform campaign sync | Meta/Google link CRUD (Enterprise SKU flag) |
| Smart alerts | Rules → Slack/Telegram via notifier |
| Margin guard | Margin policies on spend vs revenue |

Minimum Cost Sync interval: **15 minutes** (not sub-5-minute).

---

## Billing and finance

| Capability | Buyer value |
| :--- | :--- |
| Balance ledger | Postgres financial truth for customer balances |
| Invoices | PDF export, tax profiles |
| Payment webhooks | Top-up via payment providers → settlement → ledger |
| Self-serve payment intents | Advertiser wallet top-up via API |
| Disputes | Customer fraud dispute evidence (cold path) |
| Reconciliation | Postgres vs Redis spend sync, recon reports |

---

## Reporting and analytics

| Capability | Buyer value |
| :--- | :--- |
| ClickHouse analytics | Impressions, clicks, conversions, fraud events, materialized views |
| Report catalog | Geo, device, pacing, RTB, Telegram, fraud breakdown, silent reject funnel, layer desync, and more |
| Async export jobs | Large CSV/JSON exports via job polling |
| Campaign stats API | May show `stale=true` when ClickHouse lag > 5 minutes |

ClickHouse is **optional** for ingest-only dev profiles; required for IVT, ML, and most reports.

---

## Operations

| Capability | Buyer value |
| :--- | :--- |
| Outbox | Config changes apply to Redis without restarting tracker |
| DLQ | Dead-letter queue inspection and replay tools |
| Doctor / ops endpoints | Stack health, shard status |
| License | Offline JWT; apply without restart; HWID bind |
| Multi-region | Regional WAL, quorum, uplink (Enterprise / Network) |

---

## Licensing tiers (summary)

Full limits: [sku.yaml](./sku.yaml). Prices in [SALES.md](./SALES.md).

| Tier | Monthly (USDT) | Peak RPS | Hosts | IVT | ML boost | OpenRTB | Multi-region |
| :--- | ---: | ---: | ---: | :---: | :---: | :---: | :---: |
| Starter | 129 | 10k | 1 | no | no | no | no |
| Pro | 329 | 25k | 1 | yes | no | no | no |
| Scale | 649 | 75k | 3 | yes | yes | yes | no |
| Network | 1,199 | 150k | 10 | yes | yes | yes | yes |
| Enterprise | 2,500+ | custom | 99 | yes | yes | yes | yes |
| Pilot | 0 | 5k | 1 | no | no | no | no |

No license cap on campaign count or monthly events (`0` = unlimited in schema).

---

## What we do not sell

- Managed hosting or shared multi-tenant cloud
- Full ad platform UI replacement (Meta Ads Manager, etc.)
- Guaranteed sub-5-minute cost sync
- Real-time ML on every click
- OpenRTB 3.0 or multi-impression auctions
- Complete admin SPA in the current source tree (API-first until UI returns)

---

## Proof points for technical buyers

- Hot path: sub-50 ms p95 ingest target under load test; zero-alloc gate on `/track`
- Budget invariant in Postgres (`current_spend <= budget_limit`)
- Parser chaos: nginx and gnet differential count = 0
- Offline license — no phone-home license server

Operator detail: [ANTIFRAUD.md](./ANTIFRAUD.md), [docs/INTEGRATIONS.md](../../docs/INTEGRATIONS.md), [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md).
