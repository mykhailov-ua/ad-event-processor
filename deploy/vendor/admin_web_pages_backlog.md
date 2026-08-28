# Admin web pages catalog and backlog

**Sources:** `api/openapi/openapi.yaml`, `web/src/app_routes.tsx`, `web/src/models/report.ts`  
**Rules:** `ui-backlog.mdc`, `ui.mdc`, `frontend-modular.mdc`  
**Index:** `admin_ui_redesign_backlog.md` (milestone slugs)  
**Requirements:** `ADMIN_MILESTONES_REQUIREMENTS.md` (detailed requirements per milestone)

Each row is a **UI rebuild target** under `web/src/ui/<domain>/` (not legacy `components/`). Legacy page file = what exists today; migrate = rewrite per milestone spec sections 4.1–4.7.

**Status:** `legacy` = old tree ships in `web/dist`; `gap` = API exists, no dedicated route or thin stub; `pattern` = reusable milestone template first.

---

## Ship waves (recommended)

| Wave | Slug prefix | Pages | Depends on |
| :--- | :--- | :--- | :--- |
| W0 | `admin_shell`, `admin_contract_gate`, `admin_tokens` | Login, bootstrap, shell, nav | — |
| W1 | `admin_page_chrome` | PageChrome, ErrorBlock, grid primitives | W0 |
| W2 | `admin_directory_*` | Customers, campaigns list | W1 |
| W3 | `admin_detail_*` | Customer, campaign detail, invoice | W2 |
| W4 | `admin_integrations_*` | Integrations hub + subpages | W1 |
| W5 | `admin_report_*` | Reports hub + report pages | W1 |
| W6 | `admin_ops_*` | Ops home + operator tools | W1 |
| W7 | `admin_settings_*` | Settings, license, domains, team, audit | W1 |
| W8 | `admin_selfserve_*` | Buyer self-serve surfaces | W3 |
| W9 | `admin_fraud_*` | Fraud admin (if not only campaign tab) | W3 |

One page per PR unless milestone section 5 splits explicitly.

---

## W0 — Shell and auth

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/login` | shell | `POST /api/v1/session` | Email/password form; error from API; redirect to `/` | `login_page.tsx` | `ADMIN_SHELL_MILESTONE` |
| `/bootstrap` | shell | `GET/POST /api/v1/settings/platform/bootstrap` | First-run platform setup wizard | `bootstrap_page.tsx` | `ADMIN_SHELL_MILESTONE` |
| `/install/done` | shell | — | Post-install confirmation | `install_done_page.tsx` | `ADMIN_SHELL_MILESTONE` |
| `/*` (guard) | shell | `GET /api/v1/session`, `GET /api/v1/meta` | Session check; EULA gate if `eula` required; `ForbiddenPage` on RBAC | `app_boot`, `forbidden_page.tsx` | `ADMIN_SHELL_MILESTONE` |

**Shell chrome (all authenticated routes):** sidebar from `session.nav_items` or static nav map; top context; `freshness` not global — per page.

**APIs also used by shell:** `GET /api/v1/license/status`, `POST /api/v1/eula/accept`, `GET /api/v1/eula`.

---

## W1 — Home and role dashboards

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/` | dashboard | `GET /api/v1/meta`, role redirect | Operator landing: links to dashboards, incidents summary optional | `overview_page.tsx` | `ADMIN_PAGE_CHROME` + role pick |
| `/dashboards/buyer` | dashboard | `GET /api/v1/dashboards/buyer` | KPI cards, campaign health rows, `freshness_label`, stale banner | `role_dashboard_page.tsx` | `ADMIN_DASHBOARD_MILESTONE_BUYER` |
| `/dashboards/adops` | dashboard | `GET /api/v1/dashboards/adops` | Delivery/pacing KPIs | same | `ADMIN_DASHBOARD_MILESTONE_ADOPS` |
| `/dashboards/cfo` | dashboard | `GET /api/v1/dashboards/cfo` | Spend, ledger highlights | same | `ADMIN_DASHBOARD_MILESTONE_CFO` |
| `/dashboards/fraud` | dashboard | `GET /api/v1/dashboards/fraud` | Fraud KPIs, preset summary | same | `ADMIN_DASHBOARD_MILESTONE_FRAUD` |
| `/dashboards/operator` | dashboard | `GET /api/v1/dashboards/operator` | Stack health snapshot | same | `ADMIN_DASHBOARD_MILESTONE_OPERATOR` |
| `/dashboards/campaign/:id` | dashboard | `GET /api/v1/dashboards/campaign/{id}` | Single-campaign KPI strip | same | `ADMIN_DASHBOARD_MILESTONE_CAMPAIGN` |
| `/campaigns/portfolio` | dashboard | `GET /api/v1/reports/customer-portfolio` or buyer dashboard | Buyer portfolio summary | `buyer_portfolio_page.tsx` | `ADMIN_DASHBOARD_MILESTONE_BUYER` |

**Regions:** PageChrome title; freshness chip; KPI grid (server fields only); optional drill-down links.

---

## W2 — Directories

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/customers` | directory | `GET /api/v1/customers?limit&offset&sort&order` | Toolbar: link to billing; filters: sort (`name`, `created_at`), order; grid: name, balance, currency, active_campaigns, created_at; pagination. Search (`q`) deferred — not in OpenAPI yet. | `customers_page.tsx` | `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS` |
| `/campaigns` | directory | `GET /api/v1/campaigns` | Filters: customer, status, search; grid: name, status_label, budget_display, pacing, updated_at; bulk actions → `POST /api/v1/campaigns/bulk`; links to detail | `campaigns_page.tsx` | `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS` |
| `/campaigns/flows` | directory | `GET /api/v1/flows` | Flow list; link to builder | `campaign_flows_page.tsx` | `ADMIN_DIRECTORY_MILESTONE_FLOWS` |
| `/audit` | directory | `GET /api/v1/audit`, export → `GET /api/v1/audit/export` | Filter: actor, action, date range; grid: timestamp, actor, action, resource; export button | `audit_page.tsx` | `ADMIN_DIRECTORY_MILESTONE_AUDIT` |
| `/billing` | directory | `GET /api/v1/billing/invoices`, `GET /api/v1/billing/summary` | Invoice list; summary KPIs; link to invoice detail | `billing_page.tsx` | `ADMIN_DIRECTORY_MILESTONE_BILLING` |
| `/rtb/deals` | directory | `GET /api/v1/rtb/deals` | Deal grid; create deal | `rtb_deals_page.tsx` | `ADMIN_DIRECTORY_MILESTONE_RTB_DEALS` |

**Cold path:** all sort/filter/pagination via query params — no client `sortRows` on full list (`ui.mdc`).

---

## W3 — Detail and editors

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/customers/:id` | detail | `GET/PATCH /api/v1/customers/{id}`, billing sub-routes | Tabs: profile PATCH; balance `GET .../balance`; ledger `GET .../ledger`; statement, forecast, wallet, tax-profile, payments | `customer_detail_page.tsx` | `ADMIN_DETAIL_MILESTONE_CUSTOMER` |
| `/campaigns/:id` | detail | `GET/PATCH /api/v1/campaigns/{id}`, publish, stats | Sections: core PATCH fields; stats `GET .../stats`; margin; integration-health; fraud `GET/PATCH .../fraud`; publish-check/smoke buttons; events timeline `GET .../events` | `campaign_detail_page.tsx` | `ADMIN_DETAIL_MILESTONE_CAMPAIGN` |
| `/campaigns/wizard` | wizard | `POST /api/v1/campaigns/wizard/session`, onboarding-templates | Multi-step session; server-driven steps | `first_campaign_wizard_page.tsx` | `ADMIN_DETAIL_MILESTONE_WIZARD` |
| `/campaigns/migrate` | wizard | `GET .../migrate/sources`, preview/import, pull preview/import | Source picker; preview table; import confirm | `campaigns_migrate_page.tsx` | `ADMIN_CAMPAIGNS_MIGRATE_MILESTONE` |
| `/campaigns/flows/:id/builder` | editor | `GET/PATCH /api/v1/flows/{id}`, validate | Flow graph / path editor | `flow_builder_page.tsx` | `ADMIN_DETAIL_MILESTONE_FLOW_BUILDER` |
| `/campaigns/landers/:id/editor` | editor | `GET .../hosted-editor`, hosted-files, publish | ZIP/tree editor; hosted file CRUD | `lander_editor_page.tsx` | `ADMIN_DETAIL_MILESTONE_LANDER` |
| `/campaigns/:id/telegram` | detail | `GET/POST /api/v1/telegram/bots`, postbacks, deeplink-tokens | Bot list; webhook URL; postback test | `campaign_telegram_page.tsx` | `ADMIN_DETAIL_MILESTONE_TELEGRAM` |
| `/billing/invoices/:id` | detail | `GET /api/v1/billing/invoices/{id}`, pdf, ledger-lines, deliveries | Invoice header; line grid; PDF download; delivery retry | `invoice_detail_page.tsx` | `ADMIN_DETAIL_MILESTONE_INVOICE` |
| `/rtb/integration` | detail | `GET/PATCH /api/v1/rtb/integration-profile`, shadow-diff, validate-bid-request | Profile form; shadow diff table; bid validator | `rtb_integration_page.tsx` | `ADMIN_DETAIL_MILESTONE_RTB` |

**Related campaign APIs (tabs or sub-panels):** `conversion-mappings`, `placement-blocks`, `macro-preview`, `diff`, `clone`, `export/import`, `forecast/campaign`.

---

## W4 — Integrations hub

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/integrations/cost-sync` | hub | `GET /api/v1/cost-sync/networks`, credentials, run, history | Network catalog; credential forms per network; manual run; history grid | `integrations_cost_sync_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_COST_SYNC` |
| `/integrations/postbacks` | hub | `GET/PUT /api/v1/postbacks/config`, dlq, campaign-status, test | Per-campaign CAPI config; DLQ retry; dry-run test | `integrations_postbacks_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_POSTBACKS` |
| `/integrations/schemas` | directory | `GET/POST /api/v1/integration/schemas`, apply | Schema list; create; apply to campaign | `integrations_schemas_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_SCHEMAS` |
| `/integration/templates/import` | detail | `GET /api/v1/integration/templates`, import | Template catalog; import bundle | `integrations_templates_import_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_TEMPLATES` |
| `/integrations/supply` | hub | sellers, ads-txt, preview, validation, export-path | sellers.json CRUD; ads.txt CRUD; validation panel | `integrations_supply_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_SUPPLY` |
| `/integrations/margin-guard`, `/margin-guard` | hub | `GET/POST /api/v1/margin-guard/policies`, activity, overrides | Policy list; override form; activity log | `integrations_margin_guard_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_MARGIN_GUARD` |
| `/integrations/smart-alerts`, `/smart-alerts` | hub | rules CRUD, history, ack | Rule editor; fired events; ack | `integrations_smart_alerts_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_SMART_ALERTS` |
| `/integrations/automation` | hub | presets, rules CRUD, dry-run | Automation rules; preset picker; dry-run result | `integrations_automation_page.tsx` | `ADMIN_INTEGRATIONS_MILESTONE_AUTOMATION` |

**Gap — API only (no dedicated route today):** `platform-campaigns/*` (pause/resume/budget/sync) — embed in campaign detail or new `/integrations/platform-campaigns`.

---

## W5 — Reports

**Hub:** `/reports` → `GET /api/v1/reports/catalog`; cards from catalog; `live: true` only when route + API exist.

**Shared report page regions:** PageChrome; date range `from`/`to` in URL; optional `compare`; FilterPanel (campaign_id, customer_id per OpenAPI); Content = grid or heatmap; `freshness_label`, `stale`; export → `POST /api/v1/reports/jobs` + poll `GET .../jobs/{id}`.

| Route | Report key | Primary API | Content summary | Legacy | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/reports/placements` | placements | `GET /api/v1/reports/placements` | Zone/placement grid, spend, ROI, IVT | dedicated page | `ADMIN_REPORT_MILESTONE_PLACEMENTS` |
| `/reports/keywords` | keywords | `GET .../keywords` | Keyword grid | dedicated | `ADMIN_REPORT_MILESTONE_KEYWORDS` |
| `/reports/pacing-drift` | pacing-drift | `GET .../pacing-drift` | Campaign pacing drift | dedicated | `ADMIN_REPORT_MILESTONE_PACING_DRIFT` |
| `/reports/filter-rejects` | filter-rejects | `GET .../filter-rejects` | Reject reason breakdown | dedicated | `ADMIN_REPORT_MILESTONE_FILTER_REJECTS` |
| `/reports/fraud-breakdown` | fraud-breakdown | `GET .../fraud-breakdown` | Fraud type breakdown | dedicated | `ADMIN_REPORT_MILESTONE_FRAUD_BREAKDOWN` |
| `/reports/silent-reject-impression-funnel` | silent-reject-impression-funnel | `GET .../silent-reject-impression-funnel` | Funnel steps (canonical name) | dedicated | `ADMIN_REPORT_MILESTONE_SILENT_REJECT_FUNNEL` |
| `/reports/ghost-impression-funnel` | alias | same | Legacy URL alias → canonical report | alias route | (same milestone) |
| `/reports/spend-velocity` | spend-velocity | `GET .../spend-velocity` | Time series / velocity | dedicated | `ADMIN_REPORT_MILESTONE_SPEND_VELOCITY` |
| `/reports/daypart-heatmap` | daypart-heatmap | `GET .../daypart-heatmap` | Heatmap grid | dedicated | `ADMIN_REPORT_MILESTONE_DAYPART` |
| `/reports/campaign-geo-device` | campaign-geo-device | `GET .../campaign-geo-device` | Geo × device grid | dedicated | `ADMIN_REPORT_MILESTONE_GEO_DEVICE` |
| `/reports/geo-roi` | geo-roi | `GET .../geo-roi` | Geo ROI table | dedicated | `ADMIN_REPORT_MILESTONE_GEO_ROI` |
| `/reports/source-quality` | source-quality | `GET .../source-quality` | Sub-source quality | dedicated | `ADMIN_REPORT_MILESTONE_SOURCE_QUALITY` |
| `/reports/ivt-by-source` | ivt-by-source | `GET .../ivt-by-source` | IVT by source | dedicated | `ADMIN_REPORT_MILESTONE_IVT` |
| `/reports/clicks` | click-log | `GET .../click-log` or `.../clicks` | Click log directory | `click_log_page` | `ADMIN_REPORT_MILESTONE_CLICK_LOG` |
| `/reports/conversion-type-payout` | conversion-type-payout | `GET .../conversion-type-payout` | Payout by conversion type | dedicated | `ADMIN_REPORT_MILESTONE_CONV_PAYOUT` |
| `/reports/postback-reconciliation` | postback-reconciliation | `GET .../postback-reconciliation` | Postback recon grid | dedicated | `ADMIN_REPORT_MILESTONE_POSTBACK_RECON` |
| `/reports/rtb/overview` | rtb-overview | `GET .../reports/rtb/overview` | RTB KPIs | dedicated | `ADMIN_REPORT_MILESTONE_RTB_OVERVIEW` |
| `/reports/rtb/no-bid-reasons` | rtb-no-bid-reasons | `GET .../rtb/no-bid-reasons` | No-bid reasons | dedicated | `ADMIN_REPORT_MILESTONE_RTB_NO_BID` |
| `/reports/rtb/geo-device` | rtb-geo-device | `GET .../rtb/geo-device` | RTB geo/device | dedicated | `ADMIN_REPORT_MILESTONE_RTB_GEO` |
| `/reports/traffic-sources` | traffic-sources | `GET .../traffic-sources` | Source funnel | dedicated | `ADMIN_REPORT_MILESTONE_TRAFFIC_SOURCES` |
| `/reports/discrepancy-buy-sell` | discrepancy-buy-sell | `GET .../discrepancy-buy-sell` | Buy vs sell | dedicated | `ADMIN_REPORT_MILESTONE_DISCREPANCY` |
| `/reports/true-roi` | true-roi | `GET .../true-roi` | True ROI table | dedicated | `ADMIN_REPORT_MILESTONE_TRUE_ROI` |
| `/reports/cost-sync-coverage` | cost-sync-coverage | `GET .../cost-sync-coverage` | Cost sync gaps | dedicated | `ADMIN_REPORT_MILESTONE_COST_SYNC_COV` |
| `/reports/campaign-overview` | campaign-overview | `GET .../campaign-overview` | Multi-campaign summary | dedicated | `ADMIN_REPORT_MILESTONE_CAMPAIGN_OVERVIEW` |
| `/reports/customer-portfolio` | customer-portfolio | `GET .../customer-portfolio` | Customer-level portfolio | dedicated | `ADMIN_REPORT_MILESTONE_CUSTOMER_PORTFOLIO` |
| `/reports/data-quality` | data-quality | `GET .../data-quality` | CH quality signals | dedicated | `ADMIN_REPORT_MILESTONE_DATA_QUALITY` |
| `/reports/telegram` | telegram | `GET .../reports/telegram/summary` | TG summary KPIs | telegram shell | `ADMIN_REPORT_MILESTONE_TG_SUMMARY` |
| `/reports/telegram/funnel` | — | `GET .../telegram/funnel` | Funnel | dedicated | `ADMIN_REPORT_MILESTONE_TG_FUNNEL` |
| `/reports/telegram/bots` | — | `GET .../telegram/bots` | Bot stats | dedicated | `ADMIN_REPORT_MILESTONE_TG_BOTS` |
| `/reports/telegram/premium` | — | `GET .../telegram/premium` | Premium users | dedicated | `ADMIN_REPORT_MILESTONE_TG_PREMIUM` |
| `/reports/telegram/fraud` | — | `GET .../telegram/fraud` | TG fraud | dedicated | `ADMIN_REPORT_MILESTONE_TG_FRAUD` |
| `/reports/:reportKey` | dynamic | per catalog | **Stub** for keys without dedicated page — must not claim `live` | `report_stub_page.tsx` | Close stubs per report |

**Reports in OpenAPI without dedicated SPA route (gap — add route or fold into hub):**

| API path | Suggested route | Content |
| :--- | :--- | :--- |
| `GET /api/v1/reports/layer-desync-summary` | `/reports/layer-desync-summary` | Layer mismatch summary table |
| `GET /api/v1/reports/layer-desync-drilldown` | `/reports/layer-desync-drilldown` | Drilldown by fraud_reason |
| `GET /api/v1/reports/fraud-evidence-pack` | `/reports/fraud-evidence-pack` | Evidence export grid |
| `GET /api/v1/reports/signal-effectiveness` | `/reports/signal-effectiveness` | Signal hit rates |
| `GET /api/v1/reports/rtt-split-tunnel` | `/reports/rtt-split-tunnel` | RTT split tunnel |
| `GET /api/v1/reports/campaign-toggle-cohort` | `/reports/campaign-toggle-cohort` | Toggle cohort |
| `GET /api/v1/reports/wire-signal-breakdown` | `/reports/wire-signal-breakdown` | Wire signal breakdown |
| `GET /api/v1/reports/customer-fraud-*` | `/reports/customer-fraud/...` | Customer fraud suite |
| `GET /api/v1/reports/ml/*` | `/reports/ml/...` | ML shadow/score reports |
| `GET /api/v1/reports/edge-parity` | `/reports/edge-parity` | Edge parity (also `/ops/edge-parity`) |

**Saved views:** `GET/POST /api/v1/views` — filter preset bar on report pages, not standalone page.

**Schedules:** `GET/POST /api/v1/report-schedules` — **gap:** needs `/settings/report-schedules` or integrations subpage.

---

## W6 — Ops

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/ops` | dashboard | `GET /api/v1/ops/dashboard/summary`, metrics, stream | Incident cards; metric tiles; optional SSE stream | `ops_home_page.tsx` | `ADMIN_OPS_MILESTONE_HOME` |
| `/ops/shards` | detail | `GET /api/v1/ops/shards`, catchup | Slot map; migration status; catchup trigger | `ops_shards_page.tsx` | `ADMIN_OPS_MILESTONE_SHARDS` |
| `/ops/dlq` | directory | `GET /api/v1/ops/dlq`, inbox, retry | DLQ grid; inbox; retry actions | `ops_dlq_page.tsx` | `ADMIN_OPS_MILESTONE_DLQ` |
| `/ops/domains` | detail | `GET /api/v1/ops/domains/rotation`, tls-allowed | Rotation policy; per-host TLS allow | `ops_domain_rotation_page.tsx` | `ADMIN_OPS_MILESTONE_DOMAINS` |
| `/ops/blacklist` | directory | `GET/POST /api/v1/ops/blacklist` | IP blacklist CRUD | `ops_blacklist_page.tsx` | `ADMIN_OPS_MILESTONE_BLACKLIST` |
| `/ops/recon` | directory | `GET /api/v1/recon/runs` | Recon run history | `ops_recon_page.tsx` | `ADMIN_OPS_MILESTONE_RECON` |
| `/ops/consent` | directory | `GET /api/v1/ops/consent/proofs`, `GET /api/v1/consent` | Consent proof list | `ops_consent_page.tsx` | `ADMIN_OPS_MILESTONE_CONSENT` |
| `/ops/ml-model` | detail | `GET /api/v1/ops/ml-model`, eval, labels | Model status; eval run; label import | `ops_ml_model_page.tsx` | `ADMIN_OPS_MILESTONE_ML` |
| `/ops/edge-parity` | report | `GET /api/v1/reports/edge-parity` | Parser parity metrics | `ops_edge_parity_page.tsx` | `ADMIN_OPS_MILESTONE_EDGE_PARITY` |

**Ops APIs without dedicated page (embed in `/ops` or sub-routes):** `ops/doctor`, `ops/outbox`, `ops/incidents`, `ops/health/snapshot`, `ops/rum`, `ops/roles/reload`, `fraud/presets` ops path.

---

## W7 — Settings, team, support

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/settings` | detail | `GET/PATCH /api/v1/settings/platform`, apply | Platform knobs form; apply triggers outbox | `settings_page.tsx` | `ADMIN_SETTINGS_MILESTONE_PLATFORM` |
| `/settings/license` | detail | `GET /api/v1/license/status`, `POST .../apply` | JWT paste/upload; status fields | `settings_license_page.tsx` | `ADMIN_SETTINGS_MILESTONE_LICENSE` |
| `/settings/domains` | directory | `GET /api/v1/domains`, park, probe, ssl/setup | Domain grid; park; probe; SSL setup | `settings_domains_page.tsx` | `ADMIN_SETTINGS_MILESTONE_DOMAINS` |
| `/team` | hub | `GET /api/v1/team/overview`, members, budget-approvals | Members grid; invite; approval queue | `team_page.tsx` | `ADMIN_SETTINGS_MILESTONE_TEAM` |
| `/support/feedback` | detail | `GET .../support/feedback/meta`, `POST .../feedback` | Feedback form; category from meta | `support_feedback_page.tsx` | `ADMIN_SETTINGS_MILESTONE_SUPPORT` |

**Gap pages (API exists):**

| Suggested route | API | Content |
| :--- | :--- | :--- |
| `/settings/disputes` | `GET /api/v1/disputes` | Chargeback/dispute grid |
| `/settings/report-schedules` | `GET/POST /api/v1/report-schedules` | Scheduled report CRUD |
| `/brands` | `GET/POST /api/v1/brands`, creatives | Brand/creative admin (currently no route) |

---

## W8 — Self-serve and publisher

| Route | Pattern | Primary API | Page content | Legacy file | Milestone |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `/selfserve` | dashboard | selfserve portfolio APIs | Buyer home | `selfserve_portfolio_page.tsx` | `ADMIN_SELFERVE_MILESTONE_HOME` |
| `/selfserve/billing` | detail | `GET .../selfserve/billing/statement`, invoices | Statement; invoices | `selfserve_billing_page.tsx` | `ADMIN_SELFERVE_MILESTONE_BILLING` |
| `/selfserve/api-keys` | directory | `GET/POST .../selfserve/api-keys` | API key list; create (raw_key once) | `selfserve_api_keys_page.tsx` | `ADMIN_SELFERVE_MILESTONE_KEYS` |
| `/selfserve/campaigns/new` | wizard | `POST .../selfserve/campaigns`, templates | Quick campaign create | `selfserve_campaign_create_page.tsx` | `ADMIN_SELFERVE_MILESTONE_CREATE` |
| `/publisher` | dashboard | `GET /api/v1/publisher/dashboard`, statements | Publisher KPIs; statements list | `publisher_page.tsx` | `ADMIN_PUBLISHER_MILESTONE` |

---

## W9 — Fraud admin (API-first)

No top-level `/fraud/*` routes in `app_routes.tsx` today. Surfaces:

| Surface | API | Content |
| :--- | :--- | :--- |
| Campaign tab | `GET/PATCH /api/v1/campaigns/{id}/fraud`, preview | Per-campaign fraud config |
| Ops / fraud dashboard | `GET /api/v1/dashboards/fraud` | KPIs |
| Dedicated admin (gap) | `GET /api/v1/fraud/decisions`, integrations, labels, overrides, presets | Decisions log; integration status; manual labels; overrides; preset editor |

**Milestone:** `ADMIN_FRAUD_MILESTONE_ADMIN` — directory + detail for decisions/labels/overrides; link from ops nav.

---

## Dev-only

| Route | Note |
| :--- | :--- |
| `/dev/components` | Component gallery — not production nav; exclude from embed QA |

---

## Cross-cutting APIs (not full pages)

| API group | UI surface |
| :--- | :--- |
| `GET/POST /api/v1/views` | Saved filter bar on report pages |
| `GET/POST /api/v1/report-schedules` | Settings subpage (gap) |
| `POST /api/v1/billing/exports` | Export job UI on billing/customer pages |
| `GET /api/v1/billing/invariant` | CFO dashboard / ops doctor widget |
| `brands`, `offers` | Campaign/brand pickers — not standalone unless milestone adds `/brands` |

---

## Per-page milestone checklist (copy into each `ADMIN_*_MILESTONE.md`)

When creating a milestone for any row above, use the matching `deploy/vendor/ADMIN_*_MILESTONE.md` (already generated). Extend **4.3** grid columns before implement; copy checklist from `ui-backlog.mdc`:

1. **4.1** — Map regions to this doc's "Page content" column  
2. **4.2** — Route + nav group from this doc  
3. **4.3** — Grid columns per domain (names in `rem`, alignment per `ui.mdc`)  
4. **4.4** — `web/src/ui/<domain>/*.module.css`  
5. **4.5** — Copy Primary API column; query params from OpenAPI  
6. **4.6** — `pages/*`, `ui/<domain>/*`, `helpers/*_api.ts`  
7. **4.7** — Pitfalls: no client sort; freshness from API; silent-reject naming  

---

## Summary counts

| Area | Routed pages | API gaps |
| :--- | ---: | ---: |
| Shell/auth | 4 | 0 |
| Dashboards | 8 | 0 |
| Directories | 6 | 0 |
| Detail/editors | 9 | 1 (platform-campaigns) |
| Integrations | 8 | 1 |
| Reports (dedicated) | 30+ | 10+ report keys |
| Ops | 9 | 5 embeddable |
| Settings/team | 5 | 3 |
| Self-serve/publisher | 5 | 0 |
| Fraud admin | 0 top-level | 1 suite |

**Total legacy page files:** 64 under `web/src/pages/` — all targets for `ui/<domain>/` migration.

---

## Related

- `admin_ui_redesign_backlog.md` — milestone index and W0–W1 gates  
- `ui-backlog.mdc` — spec shape per page  
- `api/openapi/openapi.yaml` — authoritative endpoint list  
- `web/src/app_routes.tsx` — authoritative SPA routes  
