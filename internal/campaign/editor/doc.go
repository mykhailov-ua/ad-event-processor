// Package editor registers campaign editor HTTP routes on the control plane admin API.
//
// Role:
//   - Editor UX under /api/v1/campaigns/*: shell, geo-summary, fraud-editor, validate,
//     integration-panel, macro-preview, diff, clone-preview, bulk-action, placement IVT hints.
//   - Read paths call campaign.CampaignReader (runtime.Runtime via CampaignsHTTPHandlers).
//   - POST /validate and GET /diff are dry-run only; bulk pause/resume delegates to Runtime.
//
// Topology:
//   - Blank import in internal/controlplane/register.go runs init -> campaign.SetEditorRouteRegistrar.
//   - CampaignsHTTPHandlers.Register calls registerCampaignEditorRoutes (parent route_registrars.go).
//   - Served from cmd/control admin listener (default :8188). No Postgres or outbox in this package.
//
// Invariants:
//   - Media-buyer scope: AuthorizeCampaignAccess on campaign-scoped routes when wired.
//   - Read routes accept campaigns:read or campaigns:read:masked; writes require campaigns:write.
//   - POST /validate unmarshals PatchCampaignRequest and runs validateCampaignPatch only (no PG write).
//   - POST /campaigns/bulk-action supports pause and resume only; per-id errors in response body.
//   - Margin advisories on validate are read-only (no budget mutation).
//
// Forbidden:
//   - Direct sqlc/Postgres access, outbox enqueue, or Redis writes.
//   - internal/ingest hot-path imports.
//
// Defaults and limits:
//   - bulkCampaignMaxSync: 50 campaign IDs per bulk-action request.
//   - POST bodies: pkg/coldpath.DefaultMaxBody (64 KiB).
//
// Verify:
//
//	go test ./internal/campaign/editor/ -short -run TestGetCampaignDiff -count=1
//	go test ./internal/campaign/editor/ -short -run TestPostCampaignBulk -count=1
//	go test ./internal/campaign/editor/ -short -run TestMarginAdvisoryForCampaign -count=1
package editor
