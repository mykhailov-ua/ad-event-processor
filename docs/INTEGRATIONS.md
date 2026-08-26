# Integrations

Operator-facing wiring for traffic ingest, spend import, conversion export, and supply metadata. All configuration surfaces are on the control plane (`:8188`, `/api/v1/*`) and the React admin UI under `/integrations/*`. REST shapes are documented in `api/openapi/`; new routes follow the workflow in `docs/DEVELOPMENT.md` (cost-sync pilot). Tracker: `deploy/vendor/openapi_backlog.md`.

This document states what the code ships today. It does not claim parity with cloud trackers (Voluum Automizer sub-hourly cost sync, creative upload, and so on).

---

## Admin UI routes

| UI route | API prefix | Role |
| :--- | :--- | :--- |
| `/integrations/postbacks` | `/api/v1/postbacks/*` | Outbound CAPI and webhooks per campaign |
| `/integrations/cost-sync` | `/api/v1/cost-sync/*` | Ad-network spend credentials and manual runs |
| `/integrations/schemas` | `/api/v1/integration/schemas` | Custom inbound/outbound mapping schemas |
| `/integration/templates/import` | `/api/v1/integration/templates` | Bundled YAML from `deploy/schemas/` |
| `/integrations/supply` | `/api/v1/supply/*` | `sellers.json` and `ads.txt` export |
| `/integrations/smart-alerts` | `/api/v1/smart-alerts/*` | Alert rules (Slack/Telegram via notifier) |
| `/integrations/margin-guard` | `/api/v1/margin-guard/*` | Margin policies |
| `/campaigns/:id/telegram` | `/api/v1/telegram/*` | Bot webhooks, deeplinks, Telegram postbacks |
| `/campaigns/wizard` | `/api/v1/campaigns/*` | First-campaign onboarding wizard |
| `/api/v1/platform-campaigns/*` | same | Meta/Google link CRUD and dry-run mutations (Enterprise SKU) |

---

## Inbound traffic (tracker)

Hot-path endpoints on `cmd/tracker` (`:8181-8184`):

| Endpoint | Method | Role |
| :--- | :--- | :--- |
| `/click` | GET | 302 redirect with macros; full `FilterEngine`; click budget debit |
| `/track` | POST | S2S postback / conversion ingest (JSON, protobuf, or OpenRTB 3 per campaign format) |
| `/openrtb/bid` | POST | In-process OpenRTB 2.x auction (no full filter chain) |
| `/tg/click`, `/tg/impression` | GET | Telegram Mini App traffic |

**Universal ingest:** any network can send traffic via `GET /click` and `POST /track` when destination URLs carry the right query macros (`{campaign_id}`, `{click_id}`, `{sub1}`...`{sub30}`, UTMs).

**Zero-redirect client:** `web/src/static/track.js` (`trackEvent`) POSTs to `/track` with CORS.

**Bundled click-token schemas:** 82 YAML files under `deploy/schemas/traffic_*.v1.yaml`, registered in `internal/integrationschema/catalog.go` (100 catalog entries total, including affiliate templates). Import via admin **Integration templates** or `POST /api/v1/integration/templates/import`; apply per campaign with `POST /api/v1/campaigns/{id}/apply-templates`.

Schemas map network-specific query keys to internal tokens. They do **not** pull spend from the network; use Cost Sync or pass cost macros on the click URL when the source supports it.

---

## Cost Sync (`internal/costsync`)

Daily campaign-level spend pull for ROI reports. Worker runs in `cmd/control` when `CONTROL_ENABLE_COST_SYNC=1`. Credentials are encrypted at rest (`COST_SYNC_ENCRYPTION_KEY` or `POSTBACK_ENCRYPTION_KEY`).

**Granularity:** default daily batch per network credential (campaign-level `placement_id`). Optional sub-daily sync (`sync_interval_minutes`: 15, 30, or 60) refreshes **today's** spend in Postgres `campaign_costs` and ClickHouse `cost_snapshots`, then applies click attribution in CH (`attributed_cost_micro`, `cost_source=api_token` or `api_spread`). Sub-daily runs do **not** post `balance_ledger` reconciliation; daily reconcile for yesterday remains on the hourly worker.

| Interval | Behavior |
| :--- | :--- |
| `1440` (default) | Yesterday pull on hourly tick; `reconcileCampaigns` ledger adjust |
| `60`, `30`, `15` | Today's partial day every 15 min scheduler tick (per-credential `next_run_at`); token match or spread attribution |

**Attribution config** (`token_mapping` on credential): `placement_field` (`placement_id`, `sub1`, `sub2`), `network_object` (`ad_id`, `adset_id`, `placement_id`), `attribution_mode` (`token` or `spread`). Idempotency: Postgres `cost_sync_attribution_applied` per `(sync_run_id, campaign_id, placement_id)`.

**Sub-daily network support (v1):** Facebook, Google Ads, and TikTok use the same daily insights API for today's date (partial day totals). Other networks keep daily-only until hourly adapters land.

**Not click-level on hot path:** unless the traffic source sends `{cost}` / CPC on ingest (see ingress macros below).

**UI networks** (`web/src/helpers/cost_sync_api.ts`) match `fetchNetworkCosts` in `internal/costsync/fetch.go`:

| Network ID | Auth / notes |
| :--- | :--- |
| `facebook` | OAuth refresh (`META_APP_ID`, `META_APP_SECRET` on worker) |
| `google` | OAuth refresh (`GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`) |
| `tiktok` | OAuth refresh (`TIKTOK_APP_ID`, `TIKTOK_APP_SECRET`) |
| `microsoft_ads` | Reporting API v13 async CSV; OAuth refresh; `extra_config`: `customer_id`, `developer_token` |
| `snapchat` | Ad account stats, `breakdown=campaign`; OAuth refresh (`SNAPCHAT_CLIENT_ID`, `SNAPCHAT_CLIENT_SECRET`) |
| `linkedin` | `adAnalytics`, `pivot=CAMPAIGN`, DAILY; OAuth refresh (`LINKEDIN_CLIENT_ID`, `LINKEDIN_CLIENT_SECRET`) |
| `pinterest` | Campaign list + `campaigns/analytics`; OAuth refresh (`PINTEREST_CLIENT_ID`, `PINTEREST_CLIENT_SECRET`) |
| `trafficstars` | `POST /v2/campaigns/statistics`; offline API key in `refresh_token` (exchanged via `grant_type=refresh_token`) |
| `richads` | `GET api.richads.com/api/reports/` (`segment=campaign_id` default); API key from account Settings; override `extra_config.segment` if rows are empty |
| `galaksion` | `GET ssp2-api.galaksion.com/api/v1/advertiser/statistics`; API token or `account_id` + `extra_config.password` login |
| `propellerads` | SSP API v5 `adv/statistics`; Bearer API key |
| `mgid` | `goodhits/clients/{id}/campaigns-stat`; Bearer token |
| `adsterra` | `advertiser/stats.json`; `X-API-Key` |
| `exoclick` | `statistics/a/global`; API token session login |
| `hilltopads` | `advertiser/listStats`; API key query param |
| `clickadu` | SSP `client/statistics`; API token |
| `popads` | `report_advertiser`; API key, grouped by campaign |
| `revcontent` | Stats API; `client_id` + `client_secret` with `client_credentials` Bearer refresh |
| `mondiad` | `GET api.members.mondiad.com/api/1.0/report/advertising/campaign` (`breakdown=CAMPAIGN`); OAuth client credentials (`client_id` + API key secret) with JWT refresh |
| `juicyads` | `GET api.juicyads.com/campaigns/popunders/{token}` + per-campaign advertiser stats; API token |
| `evadav` | `POST evadavapi.com/api/v2.2/advertiser/stats/campaign`; `X-Api-Key`; day filter `DD.MM.YYYY` |
| `taboola`, `outbrain` | Native ad reporting APIs |
| `tonic_rsoc`, `system1_rsoc` | RSOC feed adapters |

**Not implemented:** PopCash and other networks that only expose reporting via account manager (no stable public API in tree). RichAds public docs describe SSP `api.admachine.co` (`publisher_profit`); the advertiser path above is what RedTrack-style integrations use and is covered by httptest fixtures only until validated against a live account.

**Pop wave blocked (no public advertiser stats API as of 2026-08):** `zeropark` (campaign mgmt API only; spend via panel/export), `rollerads` (dashboard/CSV only), `pushground` and `clickadilla` (tracker integrations exist; endpoint docs are private/support), `ezmob` (reporting API docs are account-gated in the advertiser UI).

**Operator API:** `GET /api/v1/cost-sync/networks` (per-network `extra_config` field schema), `PUT /api/v1/cost-sync/credentials/{network}` (`sync_interval_minutes`, `token_mapping`), `POST /api/v1/cost-sync/run`, `GET /api/v1/cost-sync/history`. Secret `extra_config` keys are not returned on GET; response includes `extra_config_set` booleans instead.

**Ingress cost macros (optional):** Campaign `ingress_cost_config` (`PATCH /api/v1/campaigns/{id}`) selects which query param carries spend (`cost`, `cpc`, or `bid`), scale (`decimal` USD or `micro`), `max_micro` cap, and `policy` (`ignore` invalid/over-cap). Parsed cost is stored on ClickHouse `clicks.attributed_cost_micro` with `cost_source=ingress_macro`. Does not replace Cost Sync API spend; when both exist, reports prefer API-attributed cost over ingress macro (document per report).

---

## Campaign automation rules (`internal/automation`)

Cold-path worker (15 min default, `AUTOMATION_RULES_ENABLED`, `AUTOMATION_RULES_INTERVAL_MIN`) evaluates ClickHouse `placement_stats_hourly` rollups per rule. Metrics: `roi_pct`, `spend_micro`, `clicks`, `cr`, `fraud_reject_rate` (canonical; legacy alias `ivt_rate`), `silent_reject_rate`, `fraud_reject_count` (fraud metrics query `fraud_events` by `placement_id` in payload; `fraud_reject_rate` = hard fraud-stream rejects / clicks, not fraud tier IVT). Actions: `pause_campaign` (PG + outbox), `blacklist_placement` (Redis via `PAUSE_PLACEMENT` outbox), `platform_pause` (pending `platform_campaign_mutations`), `notify` (webhook). Idempotency via `automation_rule_fires.action_hash`.

Example: `fraud_reject_rate` > 25 with `blacklist_placement` blacklists zones when hard fraud-stream events exceed 25% of rolled-up clicks in the window.

API: `GET/POST /api/v1/automation/rules`, `PUT/DELETE /api/v1/automation/rules/{id}`, `POST .../dry-run`. Admin UI: `/integrations/automation`.

---

## Platform campaign sync (`internal/platformsync`)

Enterprise SKU flag `ad_platform_campaign_api` (`deploy/vendor/sku.yaml`). Cold-path worker when `CONTROL_ENABLE_PLATFORM_CAMPAIGN_SYNC=1`.

| Capability | Networks | Notes |
| :--- | :--- | :--- |
| Read-only status sync | Meta, Google Ads | Reuses Cost Sync OAuth tokens |
| Idempotent pause / resume / daily budget cap | Meta, Google Ads | Dry-run + idempotency keys on `/api/v1/platform-campaigns/*` |

No creative upload, audience editing, or bid floor changes on the platform side.

Admin UI: campaign detail tab **Platform sync** (`/campaigns/:id?tab=platform`) when JWT includes `ad_platform_campaign_api` (Enterprise).

---

## Outbound postbacks and CAPI (`internal/postback`)

Worker: `cmd/postback-sender` or in-process in control.

| Provider | Type |
| :--- | :--- |
| `facebook` | Meta Conversions API |
| `google` | Google Ads enhanced conversions |
| `tiktok` | TikTok Events API |
| `taboola` | Taboola S2S (`tblci` / `click-id`) |
| `outbrain` | Outbrain S2S (`ob_click_id`) |
| `microsoft_ads` | Microsoft Ads ApplyOfflineConversions (`msclkid`) |
| `webhook` | Generic HTTP POST |

DLQ and test dispatch: `/api/v1/postbacks/dlq`, `/api/v1/postbacks/config/{campaign_id}/test`. Fraud integration health: `/api/v1/fraud/integrations`.

---

## Affiliate templates (`deploy/schemas/affiliate_*.v1.yaml`)

77 YAML files for inbound receive postbacks, outbound lead postbacks, and status mappings (Everad, Leadbit, AdCombo, LosPollos, TerraLeads, Dr.cash, CPAmatica, Mobidea, MyLead, MaxBounty, ClickDealer, and others). Same import/apply flow as traffic schemas.

Receive templates expose a tracker URL for the affiliate network panel; they do not replace offer-side API integrations.

---

## Telegram

- Mini App ingress on tracker (`/tg/*`).
- Bot webhooks and conversion postbacks on control (`/api/v1/telegram/webhook/{bot_id}`).
- Optional Telegram CIDR allowlist on edge.

---

## Billing and ops ingress

| Source | Route | Role |
| :--- | :--- | :--- |
| Stripe | payment webhook handler | Checkout, disputes |
| Cryptomus | `/api/v1/billing/crypto/webhook` (`:8187`) | USDT top-up |
| Self-serve | `/api/v1/selfserve/payment-intents` | Customer-initiated top-up |
| Alertmanager | optional webhook | Ops alerts |
| Prometheus | metrics ports | Scrape tracker/processor/control |

Workspace-scoped usage export: `GET /api/v1/billing/usage/export`. Ledger export jobs: `POST /api/v1/billing/exports`.

---

## Explicit non-goals

- Managed OAuth app store or one-click "Sign in with Facebook" for operators
- Prebuilt CRM / ERP connectors
- Per-network click macros beyond bundled templates and custom schemas
- Sub-5-minute Cost Sync (minimum interval 15 minutes in v1)
- Full ad platform UI replacement (campaign create, creatives, policy review)

Open competitive gaps: [deploy/vendor/competitive_backlog.md](../deploy/vendor/competitive_backlog.md).
