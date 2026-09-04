# Product specification (buyer-facing)

Neutral description of **ad-event-processor** on a self-hosted install. Not a managed service. The operator runs all processes on their own servers. The vendor issues an offline license JWT (peak RPS and host limits). The vendor does not store operator campaign or event data in the default model.

**Admin UI:** HTTP API (`/api/v1/*` on port 8188) is the supported control surface. A React admin SPA exists in `web/`; a given release may ship API-only until `web/dist` is embedded. Do not represent a full browser console unless the release artifact includes it.

---

## Intended use

| Operator profile | Typical configuration |
| :--- | :--- |
| Single operator | Click URLs, S2S postbacks, rule-based traffic filters, cost import |
| Small team | Above + batch IVT rules on ClickHouse (Pro), RBAC, reports |
| Higher volume | OpenRTB bid endpoint, batch ML score overlay, multi-host (Scale+) |
| Multi-region | Regional proxy and slot migration (Network / Enterprise) |

---

## HTTP traffic surfaces

| Surface | Mechanism | Operator setup | Constraints |
| :--- | :--- | :--- | :--- |
| Click routing | `GET /click` — redirect with macros | Point ad links to click URL | Macros: `{campaign_id}`, `{click_id}`, `{sub1}`–`{sub30}`, UTMs |
| Event ingestion | `POST /track` — JSON, protobuf, or OpenRTB 3 ingress | Wire postbacks / embed `track.js` | HTTP 202 = hot-path accept; not a persistence ack |
| Browser events | `track.js` — POST `/track` with CORS | Embed on operator landers | Operator provides notice/consent as required by law |
| Telegram | `/tg/click`, `/tg/impression` | Telegram Mini App flows | Telegram-specific |
| Programmatic bids | `POST /openrtb/bid` — in-process auction | SSP/DSP integration | Scale+ license; OpenRTB 3.0 and multi-imp >1 not implemented |
| Schema templates | Bundled click schemas + affiliate YAML | Admin API for custom schemas | 82 click + 77 affiliate templates in tree |

---

## Campaign and workflow

| Capability | Scope |
| :--- | :--- |
| Campaign CRUD | Budgets, schedules, geo, segments, status, pacing |
| Flows and landers | Path builder, hosted lander routes, offers |
| Brands and creatives | Brand assignment, creative library |
| Onboarding wizard | Template-based first campaign |
| Migration import | Preview/import from Keitaro, Binom, native JSON |
| Experiments / cohorts | Cohort assignment on campaigns |
| Self-serve API keys | Advertiser API under `/api/v1/selfserve/*` |

---

## Traffic quality and measurement

Rule-based and batch layers. Not a guarantee of fraud elimination.

| Layer | Effect | Constraints |
| :--- | :--- | :--- |
| Edge (nginx) | Rate limits, circuit breaker, IP blacklist cache | Standard L7 perimeter |
| Tracker filters | Datacenter IP, time-to-click, TLS fingerprint, device mismatch, proxy heuristics | Some signals fail-open behind CDN without edge TCP/TLS sync |
| Silent reject | Decoy 202/302 instead of 403 when enabled per campaign | Analytics field: `silent_reject_event` |
| Shadow mode | Event logged; no budget debit when configured | Visible in reports only |
| IP blacklist | Redis `blacklist:fraud`; optional XDP at NIC | XDP: Enterprise license |
| IVT detector (Pro+) | Batch rules on ClickHouse | Requires CH + `analytics-ml`; not per-click real-time |
| ML fraud boost (Scale+) | Batch scores adjust thresholds via Redis | `fraud-scorer` sidecar; not on every `/track` |
| Conversion validation | Processor rejects conversions without matching click (cold path) | May defer until ClickHouse click row exists |
| Evidence export (Scale+) | Signed JSON bundle for disputes | No raw IP in bundle; signing secret required |
| Presets | `conservative`, `balanced`, `aggressive`, `enhanced_defense`, `social_in_app` | Per-campaign configuration |

Do not represent: residential proxy detection on every request, real-time ML on every click, or XDP blocking rotating proxies.

---

## Integrations

| Integration | Scope |
| :--- | :--- |
| Cost sync | Daily spend import from 25 ad networks (campaign-level) |
| Outbound CAPI | Meta, Google, TikTok conversion APIs + generic webhook |
| Postbacks | Per-campaign outbound postback configuration |
| Supply metadata | `sellers.json`, `ads.txt` export |
| Platform campaign sync | Meta/Google link CRUD (Enterprise SKU) |
| Notifications | Smart alert rules to Slack/Telegram |
| Margin guard | Margin policies on spend vs revenue |

Minimum cost sync interval: 15 minutes.

---

## Billing and finance

| Capability | Scope |
| :--- | :--- |
| Balance ledger | Postgres financial records |
| Invoices | PDF export, tax profiles |
| Payment webhooks | Top-up via payment providers |
| Self-serve payment intents | Advertiser wallet top-up via API |
| Disputes | Fraud dispute evidence export (cold path) |
| Reconciliation | Postgres vs Redis spend sync, recon reports |

---

## Reporting

| Capability | Scope |
| :--- | :--- |
| ClickHouse analytics | Impressions, clicks, conversions, fraud events, materialized views |
| Report catalog | Geo, device, pacing, RTB, Telegram, fraud breakdown, silent reject funnel, and others |
| Async export | Large CSV/JSON via job polling |
| Campaign stats API | `stale=true` when ClickHouse lag > 5 minutes |

ClickHouse is optional on `ingest-only` profiles; required for IVT, ML, and most reports.

---

## Operations

| Capability | Scope |
| :--- | :--- |
| Outbox | Config changes apply to Redis without tracker restart |
| DLQ | Dead-letter inspection and replay |
| Doctor / ops API | Stack health, shard status |
| License | Offline JWT; apply without restart; HWID bind |
| Multi-region | Regional WAL, quorum, uplink (Network / Enterprise) |

---

## License tiers

Source: [sku.yaml](./sku.yaml). Prices: [SALES.md](./SALES.md).

| Tier | Monthly (USDT) | Peak RPS | Hosts | IVT | ML boost | OpenRTB | Multi-region |
| :--- | ---: | ---: | ---: | :---: | :---: | :---: | :---: |
| Starter | 129 | 10k | 1 | no | no | no | no |
| Pro | 329 | 25k | 1 | yes | no | no | no |
| Scale | 649 | 75k | 3 | yes | yes | yes | no |
| Network | 1,199 | 150k | 10 | yes | yes | yes | yes |
| Enterprise | 2,500+ | custom | 99 | yes | yes | yes | yes |
| Pilot | 0 | 5k | 1 | no | no | no | no |

`max_active_campaigns: 0` and `max_events_per_month: 0` in SKU schema = no license cap on those fields.

---

## Out of scope

- Vendor-operated hosting or shared multi-tenant cloud
- Replacement for ad platform UIs (Meta Ads Manager, Google Ads, etc.)
- Sub-5-minute cost sync interval
- Real-time ML inference on every ingest request
- OpenRTB 3.0 or multi-impression auctions
- Guaranteed business outcomes (ROI, fraud rate, uptime on operator networks)

---

## Payment-processor business description (short)

> License and distribution of self-hosted server software for HTTP traffic measurement, event ingestion, campaign routing, and spend accounting. Software runs entirely on the customer's infrastructure. The vendor does not process or store customer end-user data in the default delivery model. Revenue is software license fees.

---

## Technical references

- [ANTIFRAUD.md](./ANTIFRAUD.md) — traffic quality behavior
- [docs/INTEGRATIONS.md](../../docs/INTEGRATIONS.md) — integration catalog
- [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md) — topology
- [PUBLIC_OFFER.md](./PUBLIC_OFFER.md) — license terms
