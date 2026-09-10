// Package teamscope resolves campaign list and mutation scope for team-bound roles.
//
// Role:
// - Centralizes MB/B ownership enforcement, TL team subset, and customer-wide admin views.
// - Called from campaign list handlers, scoped ID helpers, and AssertCampaignAccess gates.
//
// Invariants:
// - Hot path (/track, /click) must not import this package.
// - At most one PG round-trip per admin list when team_enforce_ownership or TL team_id is active.
// - Client owner_user_id query override is ignored for ScopeTeam actors without campaigns:write.
//
// Verify:
// go test ./internal/teamscope/ -short -count=1
// go test ./internal/campaign/ -short -run TestAssertCampaignAccess -count=1
package teamscope
