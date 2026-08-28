# Admin UI rebuild backlog

**Status:** `web/` restored (2026-08-28). API: `api/openapi/`. Production embed: `web/dist` via `web/embed.go`.

**Contract:** `.cursor/rules/ui.mdc`  
**Backlog spec rules:** `.cursor/rules/ui-backlog.mdc` (mandatory sections 4.1–4.7 per page)  
**Page catalog:** `admin_web_pages_backlog.md` (routes, APIs, content per page)  
**Milestone requirements (detailed):** `ADMIN_MILESTONES_REQUIREMENTS.md`  
**Milestone structure:** `deploy/vendor/MILESTONE_TEMPLATE.md` (abstract)  
**Per-milestone files:** `deploy/vendor/ADMIN_<SLUG>_MILESTONE.md` (**98 specs**; regenerate: `python3 scripts/dev/gen_admin_milestone_specs.py`)  
**Reference directory (most detailed):** `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS.md`

---

## How to use

1. Pick slug from tables below.  
2. Open matching `ADMIN_<SLUG>_MILESTONE.md` (already filled; extend page-specific 4.3 before implement).  
3. Cross-check `admin_web_pages_backlog.md` for route/API/content columns.  
4. Set status `REVIEW` before implementation PR.  
5. Paste section 7 verification in PR.

---

## Foundation milestones (ship first)

| Slug | Spec file | Depends on | Pattern | Summary |
| :--- | :--- | :--- | :--- | :--- |
| `admin_contract_gate` | `ADMIN_CONTRACT_MILESTONE_GATE.md` | — | gate | OpenAPI ↔ handlers; `openapi_gate.sh`; `openapi-types` |
| `admin_tokens` | `ADMIN_TOKENS_MILESTONE.md` | contract | shell | `tokens.css` only; rem spacing |
| `admin_shell` | `ADMIN_SHELL_MILESTONE.md` | tokens | shell | login, bootstrap, nav, session guard, embed |
| `admin_page_chrome` | `ADMIN_PAGE_CHROME_MILESTONE.md` | shell | shell | PageChrome, ErrorBlock, grid primitives, chips |

All foundation spec files: **filled (DRAFT)**.

---

## Pattern milestones (templates)

| Slug | Spec file pattern | Depends on | Use for |
| :--- | :--- | :--- | :--- |
| `admin_directory_pattern` | `ADMIN_DIRECTORY_MILESTONE_<PAGE>.md` | page_chrome | Server-paginated lists |
| `admin_detail_pattern` | `ADMIN_DETAIL_MILESTONE_<PAGE>.md` | page_chrome | GET/PATCH detail, wizards, editors |
| `admin_report_pattern` | `ADMIN_REPORT_MILESTONE_<REPORT>.md` | page_chrome | CH reports + export jobs |
| `admin_integrations_hub` | `ADMIN_INTEGRATIONS_MILESTONE_<AREA>.md` | shell | Integration subpages |

---

## Page milestones — directories (W2)

| Slug | Spec file | API ready | Route |
| :--- | :--- | :--- | :--- |
| `admin_directory_customers` | `ADMIN_DIRECTORY_MILESTONE_CUSTOMERS.md` | yes | `/customers` |
| `admin_directory_campaigns` | `ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS.md` | yes | `/campaigns` |
| `admin_directory_flows` | `ADMIN_DIRECTORY_MILESTONE_FLOWS.md` | yes | `/campaigns/flows` |
| `admin_directory_audit` | `ADMIN_DIRECTORY_MILESTONE_AUDIT.md` | yes | `/audit` |
| `admin_directory_billing` | `ADMIN_DIRECTORY_MILESTONE_BILLING.md` | yes | `/billing` |
| `admin_directory_rtb_deals` | `ADMIN_DIRECTORY_MILESTONE_RTB_DEALS.md` | yes | `/rtb/deals` |
| `admin_directory_brands` | `ADMIN_DIRECTORY_MILESTONE_BRANDS.md` | yes (route gap) | `/brands` |

Directory specs: **filled (DRAFT)**. Customers = reference (most detailed 4.3).

---

## Page milestones — detail (W3)

| Slug | Spec file | API ready | Route |
| :--- | :--- | :--- | :--- |
| `admin_detail_customer` | `ADMIN_DETAIL_MILESTONE_CUSTOMER.md` | yes | `/customers/:id` |
| `admin_detail_campaign` | `ADMIN_DETAIL_MILESTONE_CAMPAIGN.md` | yes | `/campaigns/:id` |
| `admin_detail_invoice` | `ADMIN_DETAIL_MILESTONE_INVOICE.md` | yes | `/billing/invoices/:id` |
| `admin_detail_rtb` | `ADMIN_DETAIL_MILESTONE_RTB.md` | yes | `/rtb/integration` |
| `admin_detail_flow_builder` | `ADMIN_DETAIL_MILESTONE_FLOW_BUILDER.md` | yes | `/campaigns/flows/:id/builder` |
| `admin_detail_lander` | `ADMIN_DETAIL_MILESTONE_LANDER.md` | yes | `/campaigns/landers/:id/editor` |
| `admin_detail_telegram` | `ADMIN_DETAIL_MILESTONE_TELEGRAM.md` | yes | `/campaigns/:id/telegram` |
| `admin_detail_wizard` | `ADMIN_DETAIL_MILESTONE_WIZARD.md` | yes | `/campaigns/wizard` |
| `admin_campaigns_migrate` | `ADMIN_CAMPAIGNS_MIGRATE_MILESTONE.md` | yes | `/campaigns/migrate` |

---

## Page milestones — dashboards (W1)

| Slug | Spec file | API ready | Route |
| :--- | :--- | :--- | :--- |
| `admin_dashboard_overview` | `ADMIN_DASHBOARD_MILESTONE_OVERVIEW.md` | yes | `/` |
| `admin_dashboard_buyer` | `ADMIN_DASHBOARD_MILESTONE_BUYER.md` | yes | `/dashboards/buyer` |
| `admin_dashboard_adops` | `ADMIN_DASHBOARD_MILESTONE_ADOPS.md` | yes | `/dashboards/adops` |
| `admin_dashboard_cfo` | `ADMIN_DASHBOARD_MILESTONE_CFO.md` | yes | `/dashboards/cfo` |
| `admin_dashboard_fraud` | `ADMIN_DASHBOARD_MILESTONE_FRAUD.md` | yes | `/dashboards/fraud` |
| `admin_dashboard_operator` | `ADMIN_DASHBOARD_MILESTONE_OPERATOR.md` | yes | `/dashboards/operator` |
| `admin_dashboard_campaign` | `ADMIN_DASHBOARD_MILESTONE_CAMPAIGN.md` | yes | `/dashboards/campaign/:id` |

---

## Page milestones — integrations (W4)

| Slug | Spec file | API ready | Route |
| :--- | :--- | :--- | :--- |
| `admin_integrations_hub` | `ADMIN_INTEGRATIONS_MILESTONE_HUB.md` | partial | nav hub |
| `admin_integrations_cost_sync` | `ADMIN_INTEGRATIONS_MILESTONE_COST_SYNC.md` | yes | `/integrations/cost-sync` |
| `admin_integrations_postbacks` | `ADMIN_INTEGRATIONS_MILESTONE_POSTBACKS.md` | yes | `/integrations/postbacks` |
| `admin_integrations_schemas` | `ADMIN_INTEGRATIONS_MILESTONE_SCHEMAS.md` | yes | `/integrations/schemas` |
| `admin_integrations_templates` | `ADMIN_INTEGRATIONS_MILESTONE_TEMPLATES.md` | yes | `/integration/templates/import` |
| `admin_integrations_supply` | `ADMIN_INTEGRATIONS_MILESTONE_SUPPLY.md` | yes | `/integrations/supply` |
| `admin_integrations_margin_guard` | `ADMIN_INTEGRATIONS_MILESTONE_MARGIN_GUARD.md` | yes | `/integrations/margin-guard` |
| `admin_integrations_smart_alerts` | `ADMIN_INTEGRATIONS_MILESTONE_SMART_ALERTS.md` | yes | `/integrations/smart-alerts` |
| `admin_integrations_automation` | `ADMIN_INTEGRATIONS_MILESTONE_AUTOMATION.md` | yes | `/integrations/automation` |
| `admin_integrations_platform_campaigns` | `ADMIN_INTEGRATIONS_MILESTONE_PLATFORM_CAMPAIGNS.md` | yes | gap (new route) |

---

## Page milestones — reports (W5)

One milestone per report: `ADMIN_REPORT_MILESTONE_<UPPER_SNAKE_KEY>.md`.  
Full list and API paths: `ADMIN_MILESTONES_REQUIREMENTS.md` § admin_report_pattern.

Hub: `ADMIN_REPORTS_HUB_MILESTONE.md` → `/reports` + `GET /api/v1/reports/catalog`.

**Priority reports (ship after hub):** placements, campaign-overview, true-roi, filter-rejects, fraud-breakdown, silent-reject-impression-funnel, spend-velocity, click-log.

**Gap reports (need route + milestone before `live`):** layer-desync-*, fraud-evidence-pack, ml/*, customer-fraud-*, wire-signal-breakdown, campaign-toggle-cohort, signal-effectiveness, rtt-split-tunnel.

---

## Page milestones — ops (W6)

| Slug | Spec file | Route |
| :--- | :--- | :--- |
| `admin_ops_home` | `ADMIN_OPS_MILESTONE_HOME.md` | `/ops` |
| `admin_ops_shards` | `ADMIN_OPS_MILESTONE_SHARDS.md` | `/ops/shards` |
| `admin_ops_dlq` | `ADMIN_OPS_MILESTONE_DLQ.md` | `/ops/dlq` |
| `admin_ops_domains` | `ADMIN_OPS_MILESTONE_DOMAINS.md` | `/ops/domains` |
| `admin_ops_blacklist` | `ADMIN_OPS_MILESTONE_BLACKLIST.md` | `/ops/blacklist` |
| `admin_ops_recon` | `ADMIN_OPS_MILESTONE_RECON.md` | `/ops/recon` |
| `admin_ops_consent` | `ADMIN_OPS_MILESTONE_CONSENT.md` | `/ops/consent` |
| `admin_ops_ml` | `ADMIN_OPS_MILESTONE_ML.md` | `/ops/ml-model` |
| `admin_ops_edge_parity` | `ADMIN_OPS_MILESTONE_EDGE_PARITY.md` | `/ops/edge-parity` |

---

## Page milestones — settings (W7)

| Slug | Spec file | Route |
| :--- | :--- | :--- |
| `admin_settings_platform` | `ADMIN_SETTINGS_MILESTONE_PLATFORM.md` | `/settings` |
| `admin_settings_license` | `ADMIN_SETTINGS_MILESTONE_LICENSE.md` | `/settings/license` |
| `admin_settings_domains` | `ADMIN_SETTINGS_MILESTONE_DOMAINS.md` | `/settings/domains` |
| `admin_settings_team` | `ADMIN_SETTINGS_MILESTONE_TEAM.md` | `/team` |
| `admin_settings_support` | `ADMIN_SETTINGS_MILESTONE_SUPPORT.md` | `/support/feedback` |
| `admin_settings_disputes` | `ADMIN_SETTINGS_MILESTONE_DISPUTES.md` | gap `/settings/disputes` |
| `admin_settings_report_schedules` | `ADMIN_SETTINGS_MILESTONE_REPORT_SCHEDULES.md` | gap |

---

## Page milestones — self-serve (W8)

| Slug | Spec file | Route |
| :--- | :--- | :--- |
| `admin_selfserve_home` | `ADMIN_SELFERVE_MILESTONE_HOME.md` | `/selfserve` |
| `admin_selfserve_billing` | `ADMIN_SELFERVE_MILESTONE_BILLING.md` | `/selfserve/billing` |
| `admin_selfserve_keys` | `ADMIN_SELFERVE_MILESTONE_KEYS.md` | `/selfserve/api-keys` |
| `admin_selfserve_create` | `ADMIN_SELFERVE_MILESTONE_CREATE.md` | `/selfserve/campaigns/new` |
| `admin_publisher` | `ADMIN_PUBLISHER_MILESTONE.md` | `/publisher` |

---

## Page milestones — fraud (W9)

| Slug | Spec file | Route |
| :--- | :--- | :--- |
| `admin_fraud_admin` | `ADMIN_FRAUD_MILESTONE_ADMIN.md` | gap `/fraud/*` |

Campaign fraud tab: scope inside `ADMIN_DETAIL_MILESTONE_CAMPAIGN.md`.

---

Pre-reset waves are not carried forward.

---

## Spec file status (2026-08-28)

| Area | Count | Notes |
| :--- | ---: | :--- |
| Foundation | 4 | gate, tokens, shell, page_chrome |
| Directories | 7 | customers = reference spec (full 4.3) |
| Detail + migrate | 9 | includes `ADMIN_CAMPAIGNS_MIGRATE_MILESTONE` |
| Dashboards | 7 | |
| Integrations | 10 | includes platform-campaigns gap |
| Reports hub | 1 | `ADMIN_REPORTS_HUB_MILESTONE` |
| Reports | 38 | 30 routed + 8 gap |
| Ops | 9 | |
| Settings | 7 | includes disputes + report-schedules gaps |
| Self-serve | 4 | |
| Publisher | 1 | |
| Fraud admin | 1 | |
| **Total** | **98** | ~229 lines/spec avg (CUSTOMERS hand reference ~322); regen: `python3 scripts/dev/gen_admin_milestone_specs.py` |

Catalog source: `scripts/dev/admin_milestone_catalog.py` + `admin_milestone_catalog_data.py` (OpenAPI-backed contract_rows, PATCH inventories, API gaps).
