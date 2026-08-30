// Package campaign owns campaign admin DTOs, HTTP handlers, Runtime, and publish/import orchestration.
//
// Role:
//   - HTTP: /api/v1/campaigns/* CRUD, events, margin, clone, import/export, postbacks (PostbackHTTPHandlers).
//   - Runtime (subpackage runtime): list/get/patch/publish with Effects port for outbox-heavy mutations.
//   - Subpackages register extra routes via init: editor, wizard, integration, selfserve (see each doc.go).
//   - Worker ticks (subpackage worker) started from controlplane for schedule, delivery, autoscale, drain.
//
// Topology:
//   - Host/Effects ports implemented by controlplane bridges (campaign_*_bridge.go).
//   - Imports reports/reportjob for validation jobs; must not import controlplane root (acyclic).
//   - Fraud campaign config HTTP in fraud_handlers.go; fraudadmin supplies read integrations.
//
// Invariants:
//   - Publish and budget mutations enqueue outbox_events in the same PG transaction as domain row updates.
//   - Media-buyer scope enforced via authz snapshot + AssertMediaBuyerCampaignAccess on reads/writes.
//   - Cold-path JSON bodies use pkg/coldpath limits on handlers.
//
// Forbidden:
//   - Tracker hot-path filter or UnifiedFilter imports.
//   - New campaign_* filename prefixes inside this directory (naming.mdc).
//
// Verify:
//
//	go test ./internal/campaign/ -short -count=1
//	go test ./internal/campaign/editor/ -short -count=1
package campaign
