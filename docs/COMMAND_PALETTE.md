# Command Palette (admin global search)

Engineering spec for **Ctrl+K / Cmd+K** global search and navigation in the admin SPA. Operators jump to campaigns, flows, landers, offers, reports, and integration routes without drilling through the sidebar.

Constraints: `ui.mdc`, `frontend-modular.mdc`, `react.mdc`, `cold-path.mdc`, `boundaries.mdc`, `anti-slop.mdc`, `core.mdc` (tracker ingest unchanged).

**Depends on:** `admin_shell` slug (`ui.mdc` ship order) — `web/` must exist with login, nav, and boot stub replaced. Command palette is **not** implementable on the static boot stub alone.

**Not in scope for v1:** natural-language / AI search, tracker hot-path changes, client-side full-catalog filter, search inside ClickHouse event payloads, mobile-native shortcuts outside the SPA.

---

## What it is

A **modal overlay** opened from the keyboard:

| Input | Action |
| :--- | :--- |
| `Ctrl+K` (Windows/Linux) | Open palette |
| `Cmd+K` (macOS) | Open palette |
| `Esc` | Close without navigate |
| `↑` / `↓` | Move selection |
| `Enter` | Navigate to highlighted result |
| Typing | Server-backed typeahead (debounced) |

**Result kinds (v1):**

| Kind | Example label | Navigate target |
| :--- | :--- | :--- |
| `campaign` | Campaign name + status chip | `/campaigns/{id}` |
| `flow` | Flow name | `/flows/{id}` or campaign flow tab |
| `lander` | Lander name | `/landers/{id}` |
| `offer` | Offer name | `/offers/{id}` |
| `report` | Report catalog title | `/reports/{key}` |
| `route` | Static nav entry (Integrations, Billing, …) | SPA path from catalog |
| `action` | "New campaign", "Import migration" | Route or modal trigger |

Competitor parity (Klixsor): search campaigns, offers, pages — we extend with flows, reports, and RBAC-scoped routes.

---

## Current baseline (do not re-ship as the product)

| Piece | Location | Behavior today |
| :--- | :--- | :--- |
| Admin UI | `web/` | **Removed** — boot stub only (`internal/controlplane/admin_static_stub/`) |
| Campaign list `q` | `internal/campaign/handlers.go` | Loads up to **1000** rows, filters in handler memory — **not** palette-grade |
| OpenAPI `q` on campaigns | `api/openapi/paths/campaigns.yaml` | Documented; no unified search endpoint |
| Route catalog | `internal/controlplane/routecatalog/catalog.go` | Machine list of API routes — not operator nav labels |
| Report catalog | `internal/reports/` | Server keys; no palette index API |

Command palette **must not** call `GET /api/v1/campaigns?q=…` with the 1000-row fan-out pattern. Add a dedicated cold-path search service.

---

## Architecture invariants

### Tracker hot path (`cmd/tracker`, `internal/ingest`)

| Rule | Requirement |
| :--- | :--- |
| I/O | **Zero** — palette is control-plane + browser only |
| Metrics | No new labels on `/track` or `/click` |
| Deploy | No tracker binary or nginx changes for palette |

### Control plane (`cmd/control`, `:8188`)

| Rule | Requirement |
| :--- | :--- |
| Package | New domain `internal/commandpalette/` (`handlers.go`, `store.go`, `index.go`, `doc.go`) + thin `commandpalette_bridge.go` in controlplane |
| Search | Postgres `ILIKE` / `tsvector` on indexed columns — **no** ClickHouse on typeahead path |
| Scope | Results filtered by `customer_id` + RBAC (`authz` same as list handlers) |
| Pagination | `limit` max **25** per request; `q` min length 2 after trim |
| Timeout | Handler context deadline **500 ms**; fail with partial empty + `degraded=true`, not 500 storm |
| Rate limit | Shared admin rate limiter; per-user cap **30 req/min** for search endpoint |
| Cache | Optional in-process TTL cache (5 s) keyed by `(customer_id, q_hash)` — must not bypass RBAC |

### Browser (`web/src`)

| Rule | Requirement |
| :--- | :--- |
| Module | `web/src/ui/shell/CommandPalette.tsx` + `web/src/helpers/command_palette_api.ts` |
| Cold path | Debounced fetch only; **no** `useMemo` filter over full entity lists (`ui.mdc`) |
| Hot UI path | Open overlay < **100 ms**; keystroke perceived < **50 ms** (`react.mdc`) |
| Scroll | Virtualized result list when > 20 rows; < **16 ms/frame** during scroll |
| A11y | `role="combobox"`, `aria-activedescendant`, focus trap while open |
| i18n | Labels from API `label` / `status_label` — no client status math |

---

## Rollout

### Contract and static route index (no search I/O)

**Goal:** Named feature, OpenAPI, static route entries, UI shell hook — palette opens with route/actions only.

| Deliverable | Detail |
| :--- | :--- |
| OpenAPI | `paths/command_palette.yaml`: `GET /api/v1/command-palette/search`, `GET /api/v1/command-palette/routes` |
| Static index | Build-time JSON from nav manifest (`web/src/nav/catalog.ts` synced with `ui.mdc` routes) served as `routes` kind |
| UI | `CommandPalette` modal in shell; Ctrl+K listener; empty `q` shows recents + static routes |
| Metrics | `command_palette_search_total`, `command_palette_search_duration_seconds`, `command_palette_open_total` |
| Doc | This file + row in `docs/DOCS.md` |

**Exit criteria**

- `bash scripts/ci/admin/openapi.sh` green for new paths.
- Stub search handler returns `{ items: [], total: 0 }` with 200.
- `web/` gate: `npm run typecheck` + `bash scripts/ci/admin/web.sh` when SPA lands.
- Holdout: `TestCommandPalette_routesCatalog_holdout` — every `live: true` nav path has a route entry.

---

### Entity search (campaigns, flows, landers, offers)

**Goal:** Typeahead across core entities for one `customer_id`.

| Deliverable | Detail |
| :--- | :--- |
| PG queries | sqlc: `SearchCampaignsByName`, `SearchFlows`, `SearchLanders`, `SearchOffers` with `LIMIT $n`, `customer_id` bind, `name ILIKE '%' || $q || '%'` or `tsvector` if index added |
| Migration | Optional `GIN` index on `lower(name)` per table — measure on 50k rows before `tsvector` |
| Handler | Merge results, rank: exact prefix > substring; tie-break `updated_at DESC` |
| RBAC | Campaign-scoped entities respect `AuthorizeCampaignAccess`; offers/landers global per customer |
| UI | Result rows with kind icon + `status_label` from DTO |

**Exit criteria**

- Integration: seed 200 campaigns → `q=foo` returns < 25 rows in < 200 ms p95 on merge-integration host.
- Holdout: `TestCommandPalette_search_crossCustomer_holdout` — customer B never sees customer A rows.
- Holdout: `TestCommandPalette_search_noThousandRowScan_holdout` — query plan or sqlmock asserts `LIMIT` in SQL (no 1000-row list fan-out).
- Fix campaign list `q` anti-pattern in **separate PR** or entity-search follow-up — do not palette-ship on 1000-row filter.

---

### Reports and integration routes

**Goal:** Search report catalog keys/titles and integration hub entries.

| Deliverable | Detail |
| :--- | :--- |
| Report index | Export stable list from report catalog (`internal/reports/catalog.go` or generated JSON at `make gen`) |
| Routes | Integrations, Cost Sync, CAPI, automation, smart alerts — `route` kind with permission gate |
| License | Hide OpenRTB / platform / multi-region entries when JWT feature off (`sku.yaml`) |
| UI | Section headers by kind; permission-denied entries omitted (not disabled lie) |

**Exit criteria**

- `report_live_routes_gate.sh` parity: searchable report = `live: true` backend.
- Holdout: `TestCommandPalette_licenseGatedRoute_holdout` — Scale-only route absent on Starter JWT.

---

### Recents, actions, and keyboard polish

**Goal:** Product feel: recent entities, quick actions, prefetch.

| Deliverable | Detail |
| :--- | :--- |
| Recents | `POST /api/v1/command-palette/recents` optional; or `localStorage` per browser (document tradeoff) |
| Actions | "New campaign", "Migration import", "Doctor" — static `action` kind |
| Prefetch | On highlight `mouseenter` / selection change, `import()` page chunk only (`react.mdc`) — no data prefetch until Enter |
| Shortcut help | `?` in palette shows shortcut sheet |

**Exit criteria**

- E2E smoke (when Playwright tier exists): open palette → type → Enter lands on campaign detail.
- Profiler: palette open + 10 keystrokes — no full-page React commit (isolated shell state).

---

### Hardening and ops

**Goal:** SLOs, abuse resistance, observability.

| Deliverable | Detail |
| :--- | :--- |
| Caps | `COMMAND_PALETTE_MAX_Q_LEN=128`; reject wider |
| Degraded mode | PG slow → return static routes only + `degraded=true` |
| Audit | Optional `command_palette_search_log` (customer_id, q length, result count — **no** raw q in prod if PII policy) |
| Load | k6 or `scripts/test/load/` admin slice: 50 concurrent search — control p95 < 300 ms |

**Exit criteria**

- `command_palette_search_duration_seconds` p95 < 300 ms in load slice.
- No regression on `pr_fast` / tracker alloc gates (palette PRs must not touch ingest).

---

## Testing requirements

### Tier honesty (`anti-slop.mdc`)

| Claim | Minimum tier | Command |
| :--- | :--- | :--- |
| Handler validation / rank unit | test-fast | `go test ./internal/commandpalette/... -short -count=1` |
| Cross-customer isolation | test-integration | `make test-integration` (`-run CommandPalette`) |
| RBAC / license gates | test-fast holdouts | Named `*_holdout` in `internal/commandpalette/` |
| UI open + debounce | admin web gate | `bash scripts/ci/admin/web.sh` (when `web/` exists) |
| Tracker unaffected | test-alloc-gate | `make test-alloc-gate` — only if ingest files touched (should be none) |

**Never cite** campaign list client-filter tests as palette search proof.

### Mandatory holdouts

| Behavior | Test name (required) | Fails if |
| :--- | :--- | :--- |
| Cross-customer leak | `TestCommandPalette_search_crossCustomer_holdout` | Foreign `customer_id` row in results |
| No 1000-row scan | `TestCommandPalette_search_noThousandRowScan_holdout` | Handler calls `ListCampaigns(..., 1000, 0)` for search |
| License-gated route hidden | `TestCommandPalette_licenseGatedRoute_holdout` | OpenRTB route on Starter license |
| Empty q server round-trip | `TestCommandPalette_emptyQuery_holdout` | `q=""` triggers PG entity search (static only) |
| Debounce storm | `TestCommandPalette_rateLimit_holdout` | > 30 searches/min per user not 429 |
| RBAC write action | `TestCommandPalette_action_requiresPermission_holdout` | "New campaign" shown without `campaigns:write` |

### Integration test contract (`integration_test_slop.sh`)

Every `commandpalette_*_integration_test.go`:

1. `testing.Short()` skip with `integration:` prefix.
2. Real `testutil.SetupPostgres`.
3. Assert result **IDs and kinds**, not only `err == nil`.
4. No `testify/mock` for PG.

### UI tests (when `web/` returns)

| Test | Assert |
| :--- | :--- |
| `CommandPalette_opensOnCtrlK` | Modal visible; focus in input |
| `CommandPalette_escCloses` | No navigation |
| `CommandPalette_serverSearch` | Mock fetch — params include `q`, not client filter |
| `CommandPalette_holdout_noClientFilter` | Removing server fetch breaks test (mutation survivor) |

### Banned tests

- Tautology `require.Equal(t, items, items)`.
- Palette test that passes when `CommandPalette.tsx` is deleted (shell must register listener).
- Claiming sub-50 ms palette search from mock fetch with 0 ms delay only — name tier.

---

## Performance and SLA

### Tracker ingest (unchanged — `core.mdc`)

Command palette work **must not** change tracker budgets.

| Metric | Ceiling | Verification |
| :--- | :--- | :--- |
| `ad_http_request_duration_seconds` p95 | < 50 ms | Unchanged on palette PRs |
| Flow select | 0 allocs/op | No ingest file changes |

### Control plane API (cold path)

| Metric | Target p95 | Hard ceiling |
| :--- | :--- | :--- |
| `GET /api/v1/command-palette/search` | < 200 ms | 500 ms (handler deadline) |
| PG per kind subquery | < 50 ms each | 4 parallel or single UNION with cap |
| Response body | < 32 KiB | `limit` <= 25 |
| Rate limit 429 | — | 30 req/min/user |

Prometheus:

| Metric | Labels | Use |
| :--- | :--- | :--- |
| `command_palette_search_total` | `kind`, `degraded` | Volume |
| `command_palette_search_duration_seconds` | — | SLO |
| `command_palette_search_errors_total` | `reason` | timeout, rate_limit |

### Browser UI (`react.mdc`)

| Surface | Target | Knob |
| :--- | :--- | :--- |
| Palette open (Ctrl+K) | < 100 ms to interactive | Lazy-load palette chunk; shell preloads after login |
| Debounced search | 200 ms debounce; perceived < 50 ms feedback | Spinner in input; keep prior results until new response |
| Result list scroll | < 16 ms/frame | Window rows; `passive` listeners |
| Keystroke while open | No full-page commit | Palette state in shell leaf `memo` |
| JSON parse | Worker if > 256 KiB | Unlikely at 25 results — document anyway |

### Postgres cost

- One round-trip per search (UNION ALL subselects or parallel goroutines with shared deadline).
- Index: `(customer_id, lower(name) text_pattern_ops)` or `GIN(to_tsvector('simple', name))` per entity table.
- No N+1 lookups for status labels — join status in search query or map in one pass.

---

## API surface (target)

| Method | Path | Role |
| :--- | :--- | :--- |
| GET | `/api/v1/command-palette/search` | `customer_id`, `q`, `limit`, `kinds[]` optional |
| GET | `/api/v1/command-palette/routes` | Static nav + report index (cacheable) |

### Search response (sketch)

```json
{
  "items": [
    {
      "id": "uuid",
      "kind": "campaign",
      "label": "US Meta Prospecting",
      "status_label": "Active",
      "status_tone": "success",
      "href": "/campaigns/uuid",
      "meta": "Updated 2h ago"
    }
  ],
  "total": 3,
  "limit": 25,
  "degraded": false,
  "freshness_label": "Just now"
}
```

Permissions: `campaigns:read` minimum for entity kinds; route kinds filtered by route permission map.

---

## UI module map (target)

```
web/src/
  ui/shell/
    CommandPalette.tsx
    CommandPalette.module.css
    use_command_palette.ts      # open state, shortcut listener
  helpers/
    command_palette_api.ts      # fetch + parse only
  nav/
    catalog.ts                  # static routes; source for routes API
```

**Forbidden:** `FilterProvider` for palette; client-side fuse.js over downloaded campaign CSV; duplicating report keys in TS without OpenAPI/catalog sync.

---

## Verification matrix (per PR)

| Touched surface | Run |
| :--- | :--- |
| `internal/commandpalette/` | `go test ./internal/commandpalette/... -short -count=1` |
| OpenAPI | `bash scripts/ci/admin/openapi.sh` |
| `web/src/ui/shell/` | `npm run typecheck`; `bash scripts/ci/admin/web.sh` |
| RBAC | `go test ./internal/controlplane/ -run CommandPalette -count=1` |
| Default merge | `bash scripts/ci/pr_fast.sh` |
| PG search claim | `make test-integration` (paste exit code) |
| Tracker regression | Confirm no `internal/ingest` diff; optional `make test-alloc-gate` if unsure |

---

## PR checklist (agents)

1. Tracker: zero diff on hot path unless explicit waiver.
2. Name verification **tier** in PR body.
3. Add holdout for cross-customer and no-1000-row-scan.
4. OpenAPI + handler + TS types in same PR when UI ships.
5. Do not claim "Ctrl+K works" from API-only PR without `web/` shell wiring.
6. Static route list must match `ui.mdc` nav — no orphan links.

---

## Related

| Doc | Topic |
| :--- | :--- |
| `ui.mdc` | Grid, cold path, `admin_shell` ship order |
| `frontend-modular.mdc` | `ui/shell/` ownership |
| `react.mdc` | Debounce, scroll budgets, prefetch |
| `cold-path.mdc` | Handler body limits, no N+1 |
| `deploy/vendor/MARKETING.md` | Buyer-facing "Command palette" when shipped |
