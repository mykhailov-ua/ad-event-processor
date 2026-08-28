// Package campaign owns campaign admin HTTP, Runtime, WizardStore, editor/migration
// routes, validators, and workers. Postgres side effects are invoked through
// controlplane Effects/DeliveryHost bridges.
//
// Import: may use controlplane/authz; must NOT import controlplane root.
//
// Verify:
//
//	go test ./internal/campaign/ -short -count=1
package campaign
