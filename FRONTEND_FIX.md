# Frontend remediation backlog (`web/`)

**Sources:** `MAIN_QA_v2.md` (manual QA, HTTP `:8188`, 2026-09-08); static style audit (grep + `frontend-slop.mdc`); verification run 2026-09-08.

**Removed from this file (verified in tree):** React #185 overlay loop (OV-*), `AppRouteErrorBoundary` + `resetKey` (RB-E*), `newRandomUuid()` in `flow_path_model.ts` (UUID-*), CSP chart swatches + `react-day-picker/style.css` (CSP-*), Campaigns Report toolbar `<Link>`, status-tab E2E with `waitForResponse` + `status=PAUSED`, P0 audit rows (domains health server filter, edge parity L2, create mutex, L1 E2E upgrades), **P1.1**–**P1.9**, **S1**–**S7**, **P2.1**–**P2.4**, **P0.2** (client + docs), **P1.10** (handler 501 holdout + script in repo).

**Honesty:** Closing any row below requires pasted command + exit code (or **not run** + named lightweight check). Banned: `'unsafe-inline'` CSP widen, Tailwind `!` on BEM leftovers, client-side full-list filter to mask missing GET, claiming green from `ui_slop.sh` alone for runtime QA rows.

---

## Verification tiers (name in PR)

| Tier | When | Commands |
| :--- | :--- | :--- |
| **Static** | Every touch | `bash scripts/ci/admin/ui_slop.sh`; `bash scripts/ci/admin/ui_surface.sh`; `read_lints` on edited paths |
| **T1 manual** | QA repro | `curl -sf http://127.0.0.1:8188/health`; DevTools Network + Console on named route |
| **T1 Playwright** | Interaction claims | `cd web && npx playwright test <spec>` with stack up |
| **Full gate** | PR merge claim | `cd web && npm run typecheck`; `bash scripts/ci/admin/web.sh` — paste exit codes |

---

## Verification run (2026-09-08)

| Command | Exit |
| :--- | :--- |
| `bash scripts/ci/admin/ui_slop.sh` | 0 |
| `bash scripts/ci/admin/ui_surface.sh` | 0 |
| `cd web && npm test` | 0 (299 tests) |
| `read_lints` on team / license / wizard load hooks | clean |
| `go test ./internal/platformadmin/domains/ -run TestDomainHealthSetupSSL_scriptMissing_holdout501` | **not run** — `go test` build failed on unrelated `internal/stream` undefined symbols (sources-only clone; needs `make gen`) |
| `npm run typecheck` / `bash scripts/ci/admin/web.sh` | **not run** |
| Playwright clone / reports CH-down | **not run** — needs stack |

**Anti-hack grep (static):**

| Check | Result |
| :--- | :--- |
| `ui-control-surface` in `button.tsx` / `admin_kit.buttonShell` | absent — `buttonShell` is Tailwind-only |
| `hover:bg-accent` in `components/ui/table.tsx` `TableCell` | absent |
| `rounded-sm,` nullification block in `app.css` | absent |
| `campaigns-filter-field` in `web/src` | absent |
| `border-r` in `campaign_list_classes.ts` th/td | absent |
| `crypto.randomUUID` in `docs_tracker_section.ts` | absent — uses `LANDER_CONVERSION_EVENT_ID_LINE` |
| Reports `catch → []` | absent in reports domains grep |

**Post-Bugbot fixes (same session):**

- Wizard: `templatesFetching` includes `revalidating` + enabled-without-data — no blank Setup on reopen.
- Team / license: post-mutation refresh uses direct `bumpRefresh`, not `useCoalescedBumpRefresh` with in-flight guard (coalescing would drop refresh after invite/apply).

---

## P0 — operator blockers (remaining)

_None — P0.1 closed 2026-09-09 (insufficient balance seed + E2E helper)._

---

## Closed (2026-09-09)

### P0.1 Campaign clone POST 400

| Field | Detail |
| :--- | :--- |
| **Root cause** | Server `insufficient balance`: customer `balance + overdraft < campaign.budget_limit` on seeded data (not client body shape) |
| **Fix** | `go run ./cmd/admin db seed-clone-balance` (also runs at end of `seed-ui` / `seed_ui_demo.sh`); E2E `fetchFirstCloneableCampaignId` + `apiMutationHeaders` (CSRF) |
| **Verify** | `go run ./cmd/admin db seed-clone-balance` → exit 0; `cd web/e2e && ADMIN_E2E_BASE_URL=http://127.0.0.1:5173 npm test -- campaign_single_clone.spec.js` → pass |

---

## Waivable / deferred (explicit)

| ID | Item | Why |
| :--- | :--- | :--- |
| **P2.5** | `report_runner_page` fetch in page | Leave until second consumer |
| **P2.6** | `ui.mdc` / `DESIGN.md` stale `.admin-page-layout`, `.admin-table--*` refs | CSS removed; docs still mention BEM hooks — cosmetic doc drift, not runtime |
| Offer delete **409** | `offer is referenced by a flow` | Correct server conflict |
| `app_sidebar.tsx` `h-9 w-9` brand mark | Icon-only (**VL-11**) |
| `truncate` on combobox option labels | Allowlisted primitives |
| Inline `style={{ width }}` on campaign `<col>` / spend bars | Data-driven (**Styling stack H**) |
| Flows RF-9 triple GET | Documented in `use_flows_page_workspace.ts` |
| `?chart_mock=1` | Labeled preview tier only |

---

## Gates checklist (operator)

```bash
# Static (default agent verification)
bash scripts/ci/admin/ui_slop.sh          # exit 0 (2026-09-08)
bash scripts/ci/admin/ui_surface.sh       # exit 0 (2026-09-08)
cd web && npm test                        # exit 0, 299 tests (2026-09-08)

# PR claim
cd web && npm run typecheck
bash scripts/ci/admin/web.sh

# T1 stack
bash scripts/dev/aed-admin up
curl -sf http://127.0.0.1:8188/health
curl -sf http://127.0.0.1:8188/api/v1/reports/catalog
curl -i http://127.0.0.1:8188/api/v1/reports/<key>?...   # 503 + JSON when CH down

# QA interaction
cd web && npx playwright test campaigns_filters.spec.js creative_flows.spec.js campaign_single_clone.spec.js

# Domains SSL
test -f scripts/install/setup_domain_ssl.sh
go test ./internal/platformadmin/domains/ -run TestDomainHealthSetupSSL_scriptMissing_holdout501 -count=1
```

Do not cite tracker `/track` p99 as admin UI SLA (`anti-slop.mdc` **Bench honesty**).

---

## Anti-patterns when fixing (do not ship)

| Banned | Why | Instead |
| :--- | :--- | :--- |
| CSP `'unsafe-inline'` for calendar/charts | Security regression | External CSS on `'self'`; `CHART_SWATCH_CLASS` |
| Tailwind `!bg-*` / `!p-0` on cells | Hides root cause | Fix owner in `app.css` or primitive |
| Client `items.filter(status)` without GET | Cold-path smuggle | URL param + `useResource` |
| `Promise.resolve([])` on error | **EH-ST1** | `ErrorBlock` / `setActionError` |
| `useCoalescedBumpRefresh` after mutations | Drops refresh when list in flight | Direct `bumpRefresh` after 2xx; coalesce manual actions only |
| Toast success before `await` 2xx | **EH-TH1** | `apiConfirmed` pattern |
| Mark QA row fixed from `ui_slop.sh` only | No runtime proof | T1 manual or Playwright L1+ |
