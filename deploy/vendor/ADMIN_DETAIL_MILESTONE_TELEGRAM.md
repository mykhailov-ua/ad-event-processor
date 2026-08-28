# ADMIN_DETAIL_MILESTONE_TELEGRAM

Detail/editor: /campaigns/:id/telegram.

**Status:** DRAFT  
**Slug:** `admin_detail_telegram`  
**Depends on:** `admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`, `ADMIN_DETAIL_MILESTONE_CAMPAIGN`  
**Blocks:** —  
**Pattern:** admin_detail_pattern  
**Domain rules:** `ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| ghost_* copy | Legacy ghost strings | Neutral Telegram copy |
| Toast before 2xx | Configure bot toast | apiConfirmed after PUT 2xx |
| Masked leak | Show bot_token to masked session | masked_gate region |
| OpenAPI drift | Invented TS fields | `make openapi-types`; fields in openapi.d.ts |
| PageChrome missing | Foundation not shipped | `test -f web/src/ui/system/page_chrome.tsx` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Client KPI math | Sum sub-resources in browser | Stats endpoints only |
| Client filter/sort on items[] | useMemo over full list | URL query params + refetch only |
| Copy legacy components/ | Reuse FilterToolbar, table_sort | New ui/<domain>/ per admin_detail_pattern |
| Silent empty on error | catch → empty table | ErrorBlock on blocking fetch |
| Flex page layout | flex on page root | CSS Grid sections per ui.mdc |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Patch legacy page in place | Smallest diff | Replace compose; ui/<domain>/ |
| `-short` only | Fast green | `admin_web.sh` + section 7 pasted |
| Skip API gap doc | Ship broken filter | Section 2 API gaps table |

### 1.4 Forbidden claims until verified

- ghost_* column headers on this page

### 1.5 Doc-only delivery

This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.

## 2. Scope

### In scope

- `/campaigns/:id/telegram` — Mini App bots; postback URLs; deeplink tokens; masked role gate
- Legacy: `campaign_telegram_page.tsx`

### Out of scope

- Client-derived KPIs
- TS-only PATCH fields

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| PUT bot fields | TelegramBot configure | campaign_id, bot_id, bot_token, webhook_url, mini_app_url, secret_token, auth_date_ttl |
| PATCH postback_url | TelegramUpdatePostbackRequest | postback_url |
| POST postback | TelegramPostback create | campaign_id, postback_url |
| POST deeplink | TelegramDeeplink | campaign_id, fbclid, ttclid, utm_* |
| POST test | POST .../telegram/postbacks/{id}/test | dry-run result |

## 4. Design spec (concrete, not intent)

### 4.1 Page inventory (what is on the page)

| Region ID | Component / section | Purpose | Data source |
| :--- | :--- | :--- | :--- |
| context | ContextBar | Campaign → Telegram | campaign id link |
| chrome | PageChrome | Telegram Mini App | static + campaign name |
| masked_gate | MaskedGate | Telegram not available for masked accounts | maskLevel permissions |
| tabs | TabBar | bots \| postbacks \| deeplink \| reports | URL ?tab= |
| tab_bots | TelegramBotsPanel | bot_token, webhook_url, mini_app_url | GET/PUT TelegramBot |
| tab_postbacks | TelegramPostbacksGrid | postback_url list + test | GET/POST/PATCH postbacks |
| tab_deeplink | TelegramDeeplinkForm | Create deeplink token | POST deeplink-tokens |
| tab_reports | TelegramReportsLink | Drill to telegram reports | static route + campaign_id |
| error | ErrorBlock | Tab fetch/mutation failure | errors |


**Tabs**

| Tab ID | Label | API |
| :--- | :--- | :--- |
| bots | Bots | GET /api/v1/telegram/bots; GET/PUT /api/v1/telegram/bots/{id} |
| postbacks | Postbacks | GET/POST /api/v1/telegram/postbacks; PATCH .../postbacks/{id} |
| deeplink | Deeplink tokens | POST /api/v1/telegram/deeplink-tokens |
| reports | Analytics | Link /reports/telegram?campaign_id={id} |

### 4.2 Route and navigation (where the page lives)

| Field | Spec |
| :--- | :--- |
| Path | `/campaigns/:id/telegram` |
| Nav group | Commercial → Campaigns → Telegram |
| Permission | campaigns:read (view); campaigns:write (configure) |
| `live` | true |

### 4.3 Layout and placement (grid contract)

### 4.4 Styles and tokens (how it looks)

| Topic | Spec |
| :--- | :--- |
| CSS ownership | `web/src/ui/telegram/*.module.css` only |
| Tokens | `var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css` |
| Page compose | Page file imports domain components; no CSS modules on page |
| Grid class | `.directory` or domain root; `.grid` on content when grid page |

### 4.5 API and cold path (what the browser does not compute)

| Concern | Handler / OpenAPI | Browser |
| :--- | :--- | :--- |
| Tabs | Separate GET per tab | No client merge of tab payloads |
| GET detail | Full DTO from handler | Render as returned |
| PATCH | Fields ⊆ OpenAPI Patch*Request | apiConfirmed after 2xx |

### 4.6 File map (where code lands)

| Path | Role |
| :--- | :--- |
| web/src/pages/campaign_telegram_page.tsx | Thin compose; tabs route optional |
| web/src/ui/telegram/telegram_detail.tsx | Detail shell |
| web/src/ui/telegram/*.module.css | Section CSS modules |
| web/src/helpers/telegram_api.ts | get/patch + sub-resource fetches |
| web/src/types/generated/openapi.d.ts | Generated types |


**Remove from this route (legacy):** ../components/campaign_telegram_section.js.

**Legacy page:** `campaign_telegram_page.tsx`

### 4.7 Pitfalls checklist (avoid documented failures)

| Pitfall | Prevention in this milestone |
| :--- | :--- |
| ghost_* UI labels | No ghost_* in telegram admin copy |
| silent_reject confusion | Telegram page ≠ fraud silent reject toggle |
| Success toast before 2xx | apiConfirmed on bot/postback save |
| TS-only PATCH fields | postback_url only on PATCH |
| Masked tab leak | Gate entire page when maskLevel=masked |
| Secret replay | bot_token shown once on create if API returns raw |
| Wrapper stack | No TelegramChrome |
| Flex page root | CSS Grid sections |
| Copy CampaignTelegramSection | ui/telegram/ modules |
| Test postback fiction | Use POST .../test endpoint only |
| Phantom PATCH bots | PUT configure per OpenAPI |
| Double chip chrome | One status chip per bot row |
| apiConfirmed on test | Test result from API only |
| Client filter/sort on `items[]` | Forbidden; only URL params in 4.5 |
| Portal filter listbox | Inline drop; wrapper width 100% |
| Double freshness chrome | FreshnessBadge in PageChrome slot only |
| Silent `catch` → empty table | ErrorBlock on list error |
| Piecemeal edit | Atomic PR: all paths in 4.6 + page replace |

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | api/openapi/paths/telegram_ops.yaml | Confirm bot + postback PATCH fields | openapi_gate.sh |
| 2 | make openapi-types | Regenerate openapi.d.ts | npm run typecheck |
| 3 | web/src/helpers/telegram_api.ts | bots, postbacks, deeplink helpers | Compiles |
| 4 | web/src/ui/telegram/campaign_telegram.tsx | Tabs + masked gate | surface gate |
| 5 | web/src/pages/campaign_telegram_page.tsx | Compose; campaign :id | <= ~120 lines |
| 6 | web/src/app_routes.tsx | Route /campaigns/:id/telegram | Resolves |
| 7 | Campaign detail link | Tab link from campaign detail | consistent campaign_id |
| 8 | Legacy cleanup | Remove CampaignTelegramSection from components/ | rg components/campaign_telegram |

## 6. SLA and performance

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| List fetch | Cold admin | N/A hot-path SLA | Not /track |
| Render | Initial paint | Skeleton in grid only | No layout shift on chrome |
| Scroll | 50 rows | < 16 ms/frame | Profiler optional |

## 7. Verification (paste in PR)

```bash
cd web && npm run typecheck
bash scripts/ci/check_ui_surface_gate.sh
bash scripts/ci/admin_web.sh
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Masked gate | Manual: masked session | gate message; no bot form |
| apiConfirmed bot | Manual: save bot config | toast after 2xx |
| No ghost labels | rg 'ghost_' web/src/ui/telegram/ | no matches |
| Reports link | Manual: analytics tab | navigates with campaign_id |
| Test postback | Manual: test button | uses POST .../test |
| ErrorBlock | Manual: block bots GET | ErrorBlock visible |


PR body must paste commands **actually run** with exit codes.

## 8. Definition of done

- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete
- [ ] G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`) satisfied
- [ ] Legacy `campaign_telegram_page.tsx` replaced or delegates to `ui/telegram/`
- [ ] Verification output pasted in PR

## 9. Rollback

Revert `web/src/ui/telegram/`, helpers, and page compose in one slice; no half migration.

