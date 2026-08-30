// Package inventory tracks controlplane composition-root drain status (file → domain mapping).
//
// Role:
//   - CompositionDrain table: which controlplane files are wiring-only vs still holding domain logic.
//   - Used by modular-monolith migration inventory tests (drain_test.go); not loaded at runtime.
//
// Invariants:
//   - Rows name concrete bridge files; Drained=true means logic moved to internal/<domain>/.
//
// Verify:
//
//	go test ./internal/controlplane/inventory/ -short -count=1
package inventory
