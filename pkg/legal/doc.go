// Package legal embeds EULA text and parses acceptance JSON stored in platform settings.
//
// Role:
//   - Text is go:embed EULA.txt; Version constant 2026-01.
//   - ParseAcceptance and FormatAcceptance back licensingadmin eula handlers.
//
// Topology:
//   - Imported by internal/licensingadmin and platformadmin bootstrap; stdlib only.
//
// Invariants:
//   - SettingsKey eula_acceptance stable in PG settings KV.
//   - Parse rejects empty or malformed JSON.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/legal/... -short -count=1
package legal
