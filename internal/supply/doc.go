// Package supply owns sellers.json, ads.txt, supply-chain validation, and on-disk export for the admin API.
//
// Role:
//   - HTTP under /api/v1/supply/* (sellers, ads-txt CRUD, preview, validation, export-path).
//   - Store persists sellers and ads.txt rows; validate.go normalizes DIRECT/RESELLER and supply-chain nodes.
//   - ExportFiles writes sellers.json and ads.txt under Host.SupplyExportPath; outbox_bridge calls it on supply file update events.
//   - AuditWorker (worker_audit.go) runs AuditCompliance every 6h from control workers.
//   - reputation.go wires optional Safe Browsing / Facebook domain checks via pkg/domainhealth.
//
// Topology:
//   - Wired via supply_admin_adapter.go; Host supplies audit logging and EnqueueSupplyFilesUpdate outbox hook.
//   - Mutations enqueue supply file refresh; export is cold-path artifact for edge/nginx serve.
//
// Invariants:
//   - ads.txt DIRECT/RESELLER lines and campaign supply chains must pass validate.go rules (MaxChainHops, ASI/SID required).
//   - sellers.json cached 60s in-process; InvalidateSellersJSONCache on mutation and export.
//   - Mutations audit with actor user id from Host.ActorUserID.
//
// Forbidden:
//   - Hot-path ingest imports; supply files are cold-path artifacts only.
//
// Verify (package has no _test.go; integration tier):
//
//	make test-integration
//	go test ./internal/controlplane/ -run TestSupplyAPI -count=1
//	go test ./internal/controlplane/ -run TestFault_SellersJSONInvalid -count=1
package supply
