// Package licenseissue issues Ed25519 JWT licenses from SKU catalog and trial registry.
//
// Role:
//   - Shared by cmd/license-issue CLI and cmd/license-vendor-api HTTP service.
//   - Signs JWT via internal/licensing; records pilot anchors in trialregistry.
//
// Verify:
// go test ./internal/licenseissue/ -short -run TestService -count=1
// go run ./cmd/license-vendor-api --help
package licenseissue
