// Package selfserve serves advertiser API-key scoped HTTP under /api/v1/selfserve/* on cmd/control.
//
// Role:
//   - Campaign create from template, template list, pause/resume, payment intents, invoice list, API key create.
//   - ValidateSelfServeAPIKeyScopes and DenyScopedAPIKeyOperatorReport gate operator-only report keys.
//   - RestrictSnapshotForAPIKeyScopes narrows authz.Snapshot to key scopes on self-serve requests.
//
// Topology:
//   - SelfServeHTTPHandlers registered in internal/controlplane/adminapi_wire_domains.go (Campaigns = Service,
//     Templates adapter, payment/billing/auth clients, selfServePerm from authz).
//   - Default admin listen :8188 (control plane). Uses parent campaign CreateCampaignSpec idempotency path.
//   - Report handlers elsewhere call DenyScopedAPIKeyOperatorReport when AuthSource is api_key.
//
// Invariants:
//   - Self-serve API key scopes subset: campaigns:read, campaigns:read:masked, campaigns:write,
//     campaigns:pause, customers:read (default campaigns:read when empty).
//   - Forbidden key scopes include audit:read, ops:write, blacklist:write, shards:read, rtb:write.
//   - POST /selfserve/campaigns requires Idempotency-Key header and template_id.
//   - Pause/resume accept campaigns:write or campaigns:pause.
//
// Forbidden:
//   - shards:write, audit-only routes, and operator report catalog keys on scoped API keys
//     (fraud-evidence-pack, filter-rejects, layer-desync-summary).
//   - internal/ingest hot-path imports.
//
// Defaults and limits:
//   - SelfServePaymentIntentMaxBody: 16384 (16 KiB, pkg/coldpath).
//   - Other POST bodies: pkg/coldpath.DefaultMaxBody (64 KiB).
//   - At most 8 scopes per API key create request.
//
// Verify:
//
//	go test ./internal/campaign/selfserve/ -short -run TestValidateSelfServeAPIKeyScopes -count=1
package selfserve
