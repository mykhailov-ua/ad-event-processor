# Admin UI full refactor

Semantic slug: `admin_ui_full_refactor`. Normative contracts: `.cursor/rules/frontend-slop.mdc`, `ui.mdc`, `frontend-modular.mdc`, `control-plane.mdc`, `anti-slop.mdc`. Detailed audit catalogs live in those rules only; this file tracks status and exit criteria in plain language.

**Status:** Implementation **closed** (2026-09-04). Operator CI exit (`ui_slop.sh`, `admin/web.sh`, Playwright on live `:8188`) still required on the merge machine.

## Purpose

Remediate admin SPA (`web/src/`) for layout honesty, RBAC, errors, mocks, refresh coalescing, and Playwright wiring proof. PRs cite concrete paths and falsify commands; subjective "looks good" is banned.

## Completed work (semantic slugs)

| Slug | Summary |
| :--- | :--- |
| `styling_stack_bem_removal` | Campaigns toolbar/table Tailwind migration; BEM removed from TSX |
| `layout_drift_campaigns` | Campaigns filter grid, table DnD/resize, control height parity |
| `layout_drift_global` | `truncate` sweep on data cells and headings (allowlist in `ui.mdc`) |
| `error_and_stub_honesty` | No fake empty lists on pages; `dev_mock` RBAC 403 parity |
| `state_coalesce_ownership` | Column prefs single owner; coalesced refresh on campaigns |
| `rbac_server_client` | Nav `permission` gates; MB ops 403 integration test |
| `test_honesty_upgrade` | `handler.test.ts` negatives; E2E read/mutation/error contracts |
| `ci_gates_green` | `ui_slop.sh` + `web.sh` green on dev (2026-09-04) |
| `request_fanout_rf` | Facets refresh scope, session caches (customers, meta, stats), team/invoice/brand fixes, `useRefreshToken` rollout |
| `shell_composition_rp` | Command palette catalog/search lanes; `CommandPaletteRow`; memo cleanup |
| `form_patch_mapping` | `campaign_editor_form.ts` patch mapping + holdout tests |
| `second_pass_catalog_audit` | Ops ML lanes, campaign ops cache, flow editor snapshot boundary, toolbar UX, fraud overrides validation, dashboard preferences Tailwind, buyer chart stub banner, brand creative weight parser |

E2E contract map: `web/e2e/README.md` (proof levels L0-L3 are defined there only, not in spec file comments).

## refactor phase exit (operator)

| Criterion | Falsify |
| :--- | :--- |
| No BEM hooks in TSX; `app.css` component BEM blocks gone | `rg 'admin-[a-z-]+__' web/src` -> 0 |
| No `truncate` on headings, KPI numbers, table data cells | `rg truncate` on `domains/`, `shell/` |
| Filter row control height via `admin_kit.controlHeight` | `bash scripts/ci/admin/ui_slop.sh` |
| No `Promise.resolve([])` fake lists on pages | `rg 'Promise\.resolve\(\[\]\)' web/src/pages` -> 0 |
| No `genericOk()` mutation fallback in `dev_mock` | `rg genericOk web/src/api/dev_mock` -> 0 |
| Campaign column prefs single persist owner | `rg saveCampaignListColumnPrefs` scoped to hook + prefs module |
| Manual refresh coalesced on campaigns owner | `useCoalescedCallback` / holdout in campaigns page |
| Nav permissions on ops, dashboards, portals | `nav_config.ts` + MB session |
| MB `/ops` deep link: API 403 + blocking UI | `ops_forbidden_mb.spec.js` or httptest |
| E2E: campaigns read, one mutation, one error with `waitForResponse` | specs listed in `web/e2e/README.md` |
| `handler.test.ts` negative rows (404, 400) | `node --test … handler.test.ts` |
| `bash scripts/ci/admin/ui_slop.sh` exit 0 | paste exit code |
| `bash scripts/ci/admin/web.sh` exit 0 | paste exit code |

### Optional (not required for slug close)

| Item | Notes |
| :--- | :--- |
| `PermissionGate` on operator routes | UX only; server RBAC still required |
| Bootstrap permissions from `authz.Snapshot` | static matrix today |
| Mobile nav `Sheet` when sidebar collapsed | overlay scroll class C |
| Full E2E L1+ on every spec | **done** for write/mutation bundle; heading-only matrix stays L0 |

## Closed regressions (do not re-open without proof)

| Area | Was | Now |
| :--- | :--- | :--- |
| Fake empty list guards | 13 pages with `Promise.resolve([])` | 0; skip uses `undefined` |
| `data ?? []` on pages | 26 pages | 0 on pages |
| Nav permissions | ops/dashboards/portals ungated | `nav_config.ts` gated |
| Column prefs dual write | table + menu both `save*` | menu delegates to owner only |
| Campaigns DnD repaint | `setState` on `dragover` | highlight on `dragStart` only |
| Campaigns BEM in TSX | 7 files | 0 `*__` in TSX |
| Handler negative tests | none | 404 / 400 rows in `handler.test.ts` |
| Mock MB 403 | none | `dev_mock/rbac.ts` + tests |
| E2E soft empty-table OR | many specs | 1 union in `campaign_editor_deep` (routing vs macro) |

## Current audit snapshot (2026-09-04 evening)

| Check | Result |
| :--- | :--- |
| `Promise.resolve([])` in pages | 0 |
| BEM `__` in TSX | 0 |
| BEM `__` in `app.css` | 0 |
| `truncate` in `web/src` | 7 (combobox/date-picker allowlist + chip CSS) |
| `genericOk` in dev_mock | 0 |
| `waitForResponse` in e2e | L1+ reads, L2 writes, L3 `ops_forbidden_mb` |
| `ui_slop.sh` | exit 0 on dev (2026-09-04) |

Tier for wiring claims: **T1** (`admin_dev=0`, live `:8188`, no mock banner).

## Verification (operator)

```bash
bash scripts/ci/admin/ui_slop.sh
bash scripts/ci/admin/web.sh
node --import ./web/scripts/test_aliases.mjs --test --experimental-strip-types web/src/api/dev_mock/handler.test.ts
go test ./internal/controlplane/ -short -run 'TestManagementAPI_RoleUserForbidden|TestFault_RBACMaskEnforced' -count=1
rg -n "admin-[a-z-]+__|Promise\.resolve\(\[\]\)|genericOk\(|\.or\(.*empty" web/src web/e2e
cd web/e2e && npx playwright test customers_list.spec.js campaigns_bulk_pause.spec.js ops_forbidden_mb.spec.js
```

## Out of scope

- Hot path / tracker ingest
- Playwright L1+ on every spec under `web/e2e/` (~30 heading-only L0 specs retained)
- Postgres RBAC schema changes (`roles.yaml` is source of truth)
- Visual redesign beyond `admin_kit` alignment

## Changelog

| Date | Event |
| :--- | :--- |
| 2026-09-04 | Initial audit; audit created |
| 2026-09-04 | Request fan-out: facets scope, customers/meta/stats caches, team roster tokens, invoice ledger guard, brand `getBrand`, refresh coalescing hook |
| 2026-09-04 | Shell: command palette lane split + row component |
| 2026-09-04 | Forms: campaign editor patch mapping holdouts; brand creative weight parser |
| 2026-09-04 | Second pass: ops ML lanes, campaign ops cache, flow snapshot boundary, fraud overrides validation, dashboard preferences Tailwind, buyer chart stub banner |
| 2026-09-04 | E2E: directory read specs + command palette; write/mutation L2 bundle; `campaign_publish`, `integrations_actions` |
| 2026-09-04 | E2E helpers: `integrationRunToken`, semantic fixture builders (no trash prefixes) |
| 2026-09-04 | refactor phase doc deduped: catalog IDs removed; contracts stay in `.cursor/rules/` |
