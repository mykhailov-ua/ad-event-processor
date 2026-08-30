// Package commandpalette serves admin global search and static navigation index (Ctrl+K).
//
// Role:
//   - HTTP: GET /api/v1/command-palette/search, /routes, /recents; POST /open, POST /recents.
//   - Service.Search merges static catalog (catalog_search.go, routes.go) with PG entity search (Store).
//   - RecentsStore persists per-user recent opens in Postgres (max 20 per user).
//   - Rank and kind filters; audit hook for palette open events when enabled.
//
// Topology:
//   - HTTPHandlers registered from controlplane; CommandPaletteService() in commandpalette_bridge.go.
//   - Store uses sqlc SearchCommandPalette* queries; per-kind budget 10 rows (perKindSearchBudget).
//   - Nav catalog: staticNavEntries plus reportNavEntries; FilterNavCatalog enforces RBAC and license features.
//   - Dedicated search rate limit via ApplyCommandPaletteSearchLimit; IP limit on routes/recents/open.
//
// Invariants:
//   - Search query length 2..128 (MinSearchQueryLen, MaxSearchQueryLen); empty query skips PG entity search.
//   - customer_id required on search and recents; cross-customer recents rejected (holdout).
//   - Entity search timeout 500 ms; PG failure returns catalog hits with degraded=true when applicable.
//   - fraud-evidence-pack nav entry hidden for masked actors.
//   - RequiredLiveNavPaths lists SPA routes that must stay live when web/ returns.
//
// Forbidden:
//   - Client-side-only search over full catalog without server query params (ui.mdc).
//   - Redis KEYS or full-table scan without per-kind LIMIT (holdout TestCommandPalette_search_noThousandRowScan_holdout).
//
// Verify:
//   go test ./internal/commandpalette/ -short -count=1
//   go test ./internal/commandpalette/ -short -run TestCommandPalette_emptyQuery_holdout -count=1
//   go test ./internal/commandpalette/ -short -run TestHTTPHandlers_listRecents_foreignCustomer_holdout -count=1
//   go test ./internal/commandpalette/ -short -run TestCommandPalette_reportSearchParity_holdout -count=1
package commandpalette
