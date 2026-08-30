// Package installer generates appliance install manifests and first-boot wiring helpers for cmd/installer.
//
// Role:
//   - Renders install.yaml templates, systemd unit snippets, and secrets.env placeholders from sku profile.
//   - Validates required paths via pkg/runtimepaths before writing /etc/ad-event-processor tree.
//
// Topology:
//   - Invoked from cmd/installer binary only; no HTTP server in package.
//
// Invariants:
//   - Install token and license paths never written world-readable.
//   - Generated compose volume names use ad_event_processor_* historical prefix.
//
// Forbidden:
//   - Runtime license verify on every template render (verify at apply only).
//
// Verify:
//
//	go test ./internal/installer/... -short -count=1
package installer
