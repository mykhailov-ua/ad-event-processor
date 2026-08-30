// Package moderatorintel verifies signed moderator IP prefix feeds for edge fraud tier hints.
//
// Role:
//   - feed_v1.go parses moderator_intel_v1 JSON lines with HMAC-SHA256 over prefix entries.
//   - VerifyFeed checks signature and expiry before nginx or control loads prefixes.
//
// Topology:
//   - Consumed by edge sync workers and fraudadmin integrations; stdlib crypto only.
//
// Invariants:
//   - Signature compare uses subtle.ConstantTimeCompare.
//   - Expired feed rejected; empty prefix list is valid no-op load.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/moderatorintel/... -short -count=1
package moderatorintel
