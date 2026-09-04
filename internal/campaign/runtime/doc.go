// Package runtime implements campaign.Runtime: Postgres-backed campaign admin logic without HTTP transport.
//
// Role:
//   - List/get/patch/publish/pause/resume/create, schedule and pacing updates, events and status history.
//   - Templates (create/list/from-template/save-as-template), clone/import/export/migration via Effects ports.
//   - Optional ClickHouseQuery for GetCampaignStats and list margin breach attachment via Effects.
//   - Implements campaign.CampaignReader; wired as Campaigns on CampaignsHTTPHandlers and campaign worker.
//
// Topology:
//   - NewRuntime(pool, effects) held on controlplane.Service; lazy init in CampaignRuntime().
//   - Mutations call campaign.Effects (Service bridges: campaign_handlers_bridge, campaign_wizard_bridge,
//     campaign_import_bridge, campaign_delivery_bridge) for outbox-heavy patches, publish, pause, clone.
//   - PG reads and scrubbing in ops.go; patch/publish transactions enqueue outbox_events through Effects
//     in the same transaction as domain row updates (not in Runtime public methods directly).
//   - HTTP routes stay in parent internal/campaign/handlers.go and subpackages (editor, selfserve, integration).
//
// Invariants:
//   - Nil Runtime, pool, or effects returns campaign.ErrServiceUnavailable on mutating entrypoints.
//   - CreateCampaign uses idempotency ledger hash and EnforceDeploymentLicenseCampaignCap before insert.
//   - PatchCampaign honors If-Match revision (see parent save_conflict tests); publish runs EvaluateCampaignPublish gate.
//   - Export/import delegate to importexport with Effects.CampaignImportExportHost().
//
// Forbidden:
//   - HTTP handlers, ServeMux registration, or direct Redis writes (Effects/outbox only).
//   - internal/ingest hot-path imports.
//
// Defaults and limits:
//   - GetCampaignStats marks stale when ClickHouse ingestion lag exceeds 5 minutes (ops.go helper).
//   - When ClickHouse is nil or hourly/daily query fails, stats fall back to PG rollups (source pg, consistency strong).
//   - Campaign list/get field scrubbing via authz.MaskLevel from context (ScrubCampaignFields).
//
// Verify:
//
//	go test ./internal/campaign/ -short -run TestPatchCampaign -count=1
//	go test ./internal/campaign/ -short -run TestEvaluatePublishBlocked -count=1
package runtime
