// Package inventory tracks controlplane composition-root drain status (file to domain mapping).
//
// Role:
//   - CompositionDrain table in drain.go: bridge files and shell files mapped to target internal/<domain>/ packages.
//   - Migration inventory for modular-monolith drain milestones; not loaded at runtime.
//
// Invariants:
//   - Each DrainRow names a concrete controlplane file and Role; Drained=true means domain logic lives in internal/<domain>/.
//   - Bridge column names the wiring file when Role is bridge wiring only.
//
// Forbidden:
//   - Importing inventory from production request paths or workers (tests and CI inventory gates only).
//
// Verify:
//
//	go test ./internal/controlplane/inventory/ -short -run TestCompositionDrainInventory_holdout -count=1
package inventory
