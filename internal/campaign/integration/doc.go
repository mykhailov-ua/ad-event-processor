// Package integration serves integration schema admin HTTP and campaign integration health checks.
//
// Role:
//   - IntegrationSchemaHTTPHandlers: CRUD /api/v1/integration/schemas*, affiliate-status-presets list,
//     bundled template catalog routes when TemplateCatalog is wired.
//   - Campaign-scoped health: GET /api/v1/campaigns/{id}/integration-health (register.go init hook).
//   - BuildCampaignIntegrationHealth evaluates traffic-template join keys, postback, ingress cost, cost-sync creds.
//
// Topology:
//   - IntegrationSchemaHTTPHandlers registered from internal/controlplane/adminapi_wire_domains.go
//     (TemplateCatalog = Service, EncryptionKey from control config).
//   - Health route registered via blank import pattern like editor (SetIntegrationHealthRegistrar).
//   - applySchema and TemplateCatalog.applyIntegrationSchema write Postgres in a single transaction;
//     tracker-visible config refresh is eventual (no outbox_events enqueue in this package).
//
// Invariants:
//   - Schema documents validated with integrationschema.ParseDocument / ValidateName before INSERT.
//   - applySchema kind switch updates campaigns, postback_configs, or status_integration_schema_id per kind.
//   - List/get schemas: campaigns:read; create/apply/import templates: campaigns:write.
//   - applyCampaignTemplates delegates to campaign.TemplateCatalog (parent misc_helpers.go).
//
// Forbidden:
//   - Outbox or Redis writes from handlers in this package.
//   - internal/ingest hot-path imports.
//
// Defaults and limits:
//   - POST/apply bodies: pkg/coldpath.DefaultMaxBody (64 KiB).
//   - Affiliate status presets are static catalog entries (read-only HTTP list).
//
// Verify:
//
//	go test ./internal/campaign/integration/ -short -run TestBuildCampaignIntegrationHealth -count=1
package integration
