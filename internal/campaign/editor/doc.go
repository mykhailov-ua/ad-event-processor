// Package editor registers campaign editor HTTP routes (shell, diff, validate, bulk actions).
//
// Role:
//   - Routes under /api/v1/campaigns/{id}/editor, /diff, /validate, /macro-preview, /bulk-action, etc.
//   - Registered via init in register.go when controlplane imports _ "internal/campaign/editor".
//   - Handlers read campaign rows and flow/fraud panels; writes go through campaign.Effects where needed.
//
// Invariants:
//   - Diff and validate endpoints are read-only or enqueue-free unless explicitly calling Effects.
//   - Masked readers use campaigns:read:masked permission on read routes.
//
// Verify:
//
//	go test ./internal/campaign/editor/ -short -count=1
package editor
