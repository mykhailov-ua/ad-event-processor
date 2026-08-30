// Package runtime implements campaign.Runtime: PG-backed list/get/patch/publish without HTTP transport.
//
// Role:
//   - Delegates mutations to campaign.Effects (controlplane Service) for outbox and validation gates.
//   - Optional ClickHouseQuery for stats-enriched list margin breach attachment.
//   - Used by controlplane campaign_handlers_bridge and tests; HTTP stays in parent campaign package.
//
// Invariants:
//   - PoolOrNil guards nil pool; callers must not bypass Effects on publish/patch budget fields.
//
// Forbidden:
//   - Direct Redis writes (Effects/outbox only).
//
// Verify:
//
//	go test ./internal/campaign/ -short -run Runtime -count=1
package runtime
