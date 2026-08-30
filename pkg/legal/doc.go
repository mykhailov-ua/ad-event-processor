// Package legal embeds EULA text and parses acceptance JSON stored in platform settings.
//
// Role:
//   - go:embed EULA.txt; Version constant (2026-01) is the current EULA revision.
//   - ParseAcceptance and MarshalAcceptance round-trip licensingadmin EULA handlers.
//   - IsCurrent reports whether stored acceptance matches Version.
//
// Topology:
//   - Imported by internal/licensingadmin (GetEulaStatus, AcceptEula) and platformadmin meta enricher.
//   - Persisted in Postgres system_settings under SettingsKey; stdlib encoding/json only.
//
// Invariants:
//   - SettingsKey eula_acceptance is stable across releases; do not rename without migration.
//   - ParseAcceptance rejects empty raw, invalid JSON, and missing version field.
//   - MarshalAcceptance emits compact JSON suitable for PG text column storage.
//   - IsCurrent is strict equality on Version; licensingadmin AcceptEula rejects mismatched version.
//
// Tradeoffs:
//   - Embedded text vs remote fetch: appliance installs must work offline; bump Version when EULA.txt changes.
//   - PG settings KV vs dedicated table: one row per deployment; licensingadmin owns write path.
//   - time.Time JSON in acceptance: RFC3339 from encoding/json; callers store UTC from AcceptEula.
//
// Forbidden:
//   - Import internal/* packages.
//   - Hot-path tracker or filter imports (admin/licensing cold path only).
//
// Verify:
//
//	go test ./pkg/legal/... -short -count=1
package legal
