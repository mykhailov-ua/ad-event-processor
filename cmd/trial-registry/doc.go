// Package main is the vendor trial registry file CLI.
//
// Role:
//   - expire-stale: mark active anchors expired when valid_until < now (optional --at RFC3339).
//   - list-pending: print open pending trial requests from JSON registry.
//   - reject-pending: reject pending id with --reason audit text.
//
// Topology:
//   - Vendor-plane JSON file (deploy/vendor/trial_registry.json); no HTTP server in this binary.
//   - Uses internal/trialregistry; pairs with cmd/license-issue and cmd/vendor-trial-bot.
//
// Invariants:
//   - Unknown command exits 2; registry IO errors exit 1.
//   - list-pending prints TSV lines to stdout; status messages on stderr.
//
// Defaults and limits:
//   - VENDOR_TRIAL_REGISTRY registry path (overridable per command with --trial-registry).
//
// Forbidden:
//   - Not customer license apply on buyer VPS (use control /api/v1/license/apply).
//   - Does not issue JWTs (use cmd/license-issue --approve-pending).
//
// Verify:
// go run ./cmd/trial-registry list-pending
// go test ./internal/trialregistry/... -short -count=1
package main
