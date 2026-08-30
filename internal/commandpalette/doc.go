// Package commandpalette serves admin global search and static navigation index (Ctrl+K).
//
// Role:
//   - HTTP: GET /api/v1/command-palette/search, /routes, /recents; POST /open, POST /recents.
//   - Search spans entities, reports, static nav entries, and actions; rank + recents in PG/Redis store.
//   - Nav catalog merges staticNavEntries with reportNavEntries; FilterNavCatalog enforces RBAC + license features.
//
// Topology:
//   - Wired as CommandPaletteHTTPHandlers in controlplane register.go.
//   - bridge in controlplane/commandpalette_bridge.go supplies catalog search deps (campaign, reports).
//   - Dedicated UserRateLimiter on search endpoint (controlplane.Handler commandPaletteLimiter).
//
// Invariants:
//   - Search query min length enforced in handlers; customer_id required on search/recents.
//   - fraud-evidence-pack nav entry hidden for masked actors.
//   - RequiredLiveNavPaths lists SPA routes that must stay live when web/ returns.
//
// Forbidden:
//   - Client-side-only search over full catalog without server query params (ui.mdc).
//
// Verify:
//
//	go test ./internal/commandpalette/ -short -count=1
package commandpalette
