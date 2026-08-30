// Package uplink forwards dedup-ready regional WAL slots to global control ingest HTTP.
//
// Role:
//   - Worker polls pkg/regionproxy/opkey Pool, POST JSON ingestRequest to Config.GlobalURL.
//   - Sets X-Admin-API-Key when Config.APIKey non-empty; honors 2xx as success.
//   - Optional BatchCommitter: PrepareForward before HTTP (quorum book + executing), Complete after forward attempt (executing -> completed).
//   - WAL TryClaimForward / MarkRemoteAcked / UnclaimForward gate at-most-once global POST per seq.
//
// Topology:
//   - Wired from cmd/region-proxy via internal/regionproxy Server.SetUplink when -global-ingest-url set.
//   - Depends on wal (payload + forward flags) and opkey (dedup slots); global handler is controlplane region ingest batch route.
//
// Invariants:
//   - forwardSlot requires WalFlagDedupReady before TryClaimForward; duplicate claims return without re-posting.
//   - Transient HTTP or claim errors retry up to ForwardMaxAttempts with linear backoff (attempt-1)*ForwardRetryBackoff.
//   - Persistent failure UnclaimForward clears WalFlagForwardClaimed so a later poll can reclaim.
//   - drainBatch skips slots when BatchCommitter PrepareForward returns !ready (increments quorumHeld); no HTTP without quorum when committer wired.
//   - loop goroutine calls runtime.LockOSThread for pinned forward work.
//
// Defaults and limits:
//   - New defaults: PollInterval 1ms, BatchSize 32, HTTPTimeout 5s, ForwardMaxAttempts 3, ForwardRetryBackoff 50ms.
//   - cmd/region-proxy sets PollInterval 1ms and BatchSize 64 when uplink enabled.
//   - drainBatch no-ops when pool nil or GlobalURL empty.
//
// Tradeoffs:
//   - HTTP uplink with bounded retries instead of broker-primary replication to global control (Enterprise multi_region profile).
//   - BatchCommitter Complete runs after forwardSlot regardless of HTTP outcome; lease completion is separate from WalFlagRemoteAcked durability.
//
// Forbidden:
//   - Not tracker /track listener; regional processor spend sync uses MULTI_REGION_ENABLED processor path, not this worker.
//
// Verify:
// go test ./pkg/regionproxy/uplink/... -short -run 'TestWorker_forwardRetriesAfterHTTPFailure|TestWorker_forwardUnclaimsOnPersistentFailure' -count=1
// go test ./tests/e2e/... -run TestE2E_RegionProxyUplink -count=1
// bash scripts/test/multi_region_resilience_drill.sh
package uplink
