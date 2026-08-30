// Package pool hosts RCU domain-pool snapshots for tracking-host rotation.
//
// Role:
//   - Table maps pool_id to active/banned hostnames for click landing and safe-page rotation.
//   - Sync polls domain_pool_domains from Postgres on interval and publishes atomic generation.
//
// Topology:
//   - Background Sync.Start goroutine; hot path reads Table snapshot via atomic load only.
//   - Used by track landing resolution and ingest handler host pickers.
//
// Forbidden:
//   - Per-request Postgres query on /track or /click accept path.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package pool
