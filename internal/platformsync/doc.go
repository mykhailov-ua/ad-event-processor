// Package platformsync mirrors linked ad-network campaigns and applies remote pause/resume/budget mutations.
//
// Role:
//   - Worker (worker.go) holds a PG advisory lock and runs 15-minute cycles: sync link statuses from vendor APIs
//     and drain platform_campaign_mutations pending rows.
//   - facebook.go, google.go, tiktok.go, microsoft_ads.go implement fetch and mutate HTTP for each network.
//   - preview.go dry-runs mutations (PreviewMutation) before platformadmin enqueues pending rows.
//   - Reuses costsync.Credential decryption and OAuth refresh via injected costsync.Worker.
//   - platformadmin campaign_sync_handlers.go exposes /api/v1/platform-campaigns/* HTTP and manual sync-run.
//
// Topology:
//   - Control worker tick only; PG platform_campaign_links and platform_campaign_mutations are source of truth.
//   - Supported networks: facebook, google, tiktok, microsoft_ads (types.go NetworkSupported).
//   - Not multi-region satellite config sync (see internal/regionproxy for enterprise uplink replication).
//
// Invariants:
//   - Remote API failure marks sync_error on the link row without implying local campaign pause succeeded
//     (TestMutationFault_remoteFailureDoesNotImplyLocalPause_holdout).
//   - PreviewMutation returns noop when external status already matches requested action.
//   - Mutation rows transition pending -> applied or failed; post-mutation link status refresh runs on success.
//   - Only one worker cycle leader per cluster (PG advisory lock platformAdvisoryLockKey).
//
// Forbidden:
//   - Hot-path tracker imports.
//   - Local campaign pause/resume without corresponding vendor write when mutation is non-noop.
//
// Verify:
//   go list -e ./internal/platformsync/
//   go test ./internal/platformsync/ -short -count=1
//   go test ./internal/platformsync/ -short -run TestPreviewMutation -count=1
//   go test ./internal/platformsync/ -short -run TestMutationFault_remoteFailureDoesNotImplyLocalPause_holdout -count=1
//   go test ./internal/platformsync/ -short -run TestMutateFacebookCampaign_pause_httptest -count=1
package platformsync
