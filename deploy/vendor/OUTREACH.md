# Outreach and buyer comms (internal)

**Audience:** sales, vendor ops. **Not shipped** to customers.

Pair with [OFFER_IMPLEMENTATION_GUIDE.md](./OFFER_IMPLEMENTATION_GUIDE.md), [MARKETING.md](./MARKETING.md), [SALES.md](./SALES.md), [VENDOR_OPS_RUNBOOK.md](./VENDOR_OPS_RUNBOOK.md).

---

## Product naming

| Surface | Name | Example |
| :--- | :--- | :--- |
| Vendor / brand | **BidShard** | "BidShard self-hosted stack", bidshard.com |
| Tracker binary (buyer-facing) | **Ad Event Processor** | "Ad Event Processor handles `/click` and `/track`" |
| Go module, JWT `product_id`, paths, env | `ad-event-processor` | `AD_EVENT_PROCESSOR_*`, `scripts/install/ad-event-processor-install.sh` |

**Buyer comms:** lead with **BidShard**; name the tracker **Ad Event Processor** (title case, not kebab-case). Use `ad-event-processor` only for install commands, repo paths, and license product id.

**Banned in outreach:** "BidShard written in Go" as if BidShard were the binary name; fake latency (3-5 ms, sub-10 ms); invented endpoint counts ("200+ APIs"); "0% event fee" (say: no per-event cap in license schema; bill peak RPS + hosts).

---

## Sales channel (year 1)

| Channel | Use |
| :--- | :--- |
| **Telegram** `@bidshardsupportbot` | Pilot, paid tiers, HWID, docs, JWT delivery, USDT tx hash |
| **bidshard.com** | Offer gate, pricing, install script (`get.sh`), architecture page |
| Bot commands | `/accept` then `/trial` (see `cmd/vendor-trial-bot`) |

**Not used for standard sales:** Zoom/calls, Calendly, ticket queues, shared Google Drive / "review folder", RFP email threads.

Network / Enterprise paid buyers may get written topology intake in Telegram; still no mandatory calls unless a separate written SOW says so ([NETWORK_ENTERPRISE_RUNBOOK.md](./NETWORK_ENTERPRISE_RUNBOOK.md)).

### Pilot intake (collect in Telegram)

| Field | Why |
| :--- | :--- |
| Telegram id | Primary identity; trial registry anchor |
| Expected peak RPS | SKU sizing |
| VPS spec (RAM/CPU) | Profile hint (`ingest-only` vs `full`) |
| Use case | Affiliate S2S, brand routing, Keitaro/Binom migration, etc. |
| HWID v2 | Before issue on `hard` bind SKUs |

**Pilot limits** ([sku.yaml](./sku.yaml)): **10 days**, **5k peak RPS**, **1 host**, rules-only antifraud (no batch IVT/ML). Offer acceptance required before issue.

**Site copy:** `deploy/marketing/site.config.json` (`pilot_days: 10`, `telegram_handle: @bidshardsupportbot`).

---

## Outreach tone

| Audience | Tone |
| :--- | :--- |
| Solo affiliate / media buyer / CPA network ops | Direct; pain = cloud tracker fees, postback gaps, silent reject, margin — match [deploy/marketing/site.config.json](../marketing/site.config.json) |
| Affiliate program / network CTO (cold email) | Short technical facts; same Telegram CTA; skip cloaker/moderator hooks unless they ask |

No enterprise SaaS cycle language ("discovery call", "schedule a demo", "customer success onboarding").

---

## Cold email template (EN)

Use as plain text. Adjust opener; do not add latency numbers or OpenRTB unless buyer asks.

```
Subject: BidShard — self-hosted Ad Event Processor (docs + pilot)

Hello,

We sell BidShard — self-hosted click routing and S2S ingest. The tracker is Ad Event Processor (Go, Redis hot path, Postgres settlement, ClickHouse optional for reports and batch antifraud). Alternative to legacy affiliate cores (Cellxpert-style click + postback).

Surfaces: GET /click, POST /track, outbound postbacks/CAPI, cost sync. Admin: OpenAPI /api/v1 on :8188.

Ingest: /track and /click do not hit Postgres per request. HTTP 202 = accepted on the tracker, not a ClickHouse write ack.

License: offline Ed25519 JWT on your box, no license ping. max_events_per_month: 0 in the schema = no per-event license cap; billable dimensions are peak RPS and host count.

Antifraud: rules on the tracker; batch IVT (Pro+) and ML boost (Scale+) are background workers, not on every request.

Data: runs on your VPS. TELEMETRY_ENABLED=false by default; vendor pulse is opt-in aggregates only.

Pilot: 10 days, 0 USDT, 1 host, 5k peak RPS, rules-only (no batch IVT/ML).

Docs / pilot: message @bidshardsupportbot with expected peak RPS and VPS spec (RAM/CPU). We send OpenAPI, architecture notes, and a pilot JWT. Install: one-line script, paste license in Admin Settings. Text only — no calls.

[Name]
BidShard
```

---

## Forbidden outreach claims

Engineering truth: [MARKETING.md](./MARKETING.md), [ANTIFRAUD.md](./ANTIFRAUD.md), `core.mdc` SLA table.

| Do not claim | Say instead |
| :--- | :--- |
| "0% event volume fee" / "flat pricing = free clicks" | No per-event cap in JWT when `max_events_per_month: 0`; license is monthly USDT by peak RPS + hosts |
| "3-5 ms" / "sub-10 ms" ingest | Do not quote ms in sales copy; internal targets are load-test/Prometheus, not a warranty |
| "200+ OpenAPI endpoints" | "OpenAPI-documented `/api/v1` control plane" |
| "ML blocks fraud on every click" | Rules on hot path; IVT/ML batch on ClickHouse (SKU-gated) |
| "Residential proxies / botnets blocked on the fly" | Name the layer (rules, intel feeds Scale+, XDP Enterprise); CDN/residential limits in [ANTIFRAUD.md](./ANTIFRAUD.md) |
| "100% private" / "we physically cannot see traffic" | Self-hosted; offline JWT; telemetry opt-in false by default; vendor does not host traffic DB in default model |
| "Pilot tailored to your RPS" | Pilot is fixed 5k RPS; paid SKU sized from stated peak RPS in Telegram |
| OpenRTB in affiliate-only pitch | Omit unless buyer runs programmatic inventory; Scale+ ([MARKETING.md](./MARKETING.md)) |

---

## Related

| File | Use |
| :--- | :--- |
| [OFFER_IMPLEMENTATION_GUIDE.md](./OFFER_IMPLEMENTATION_GUIDE.md) | FAQ scripts, payment, forbidden claims (section 9) |
| [VENDOR_OPS_RUNBOOK.md](./VENDOR_OPS_RUNBOOK.md) | Pilot issue CLI, registry |
| [deploy/marketing/](../marketing/) | Public site, offer gate, Telegram CTAs |
