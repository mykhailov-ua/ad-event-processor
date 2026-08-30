// Package conn hosts TLS fingerprint tables and residential-intel feed loaders for ingest.
//
// Role:
//   - TLSFingerprintTable RCU snapshots (JA3/JA4 block and allow lists).
//   - Residential intel feed merge for edge-parity conn-type signals on filter path.
//
// Topology:
//   - atomic.Pointer snapshots; hot readers load once per check, no locks.
//   - Reloaded from tracker background ticks or startup loaders (not per-request PG).
//
// Forbidden:
//   - Synchronous network fetch on Tier B /track request path.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package conn
