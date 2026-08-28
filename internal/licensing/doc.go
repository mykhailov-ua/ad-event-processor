// Package licensing: JWT verify, entitlements, trial tier gates. Tracker reads license
// snapshot from file; no per-request license HTTP on hot path.
//
// Verify:
//   make license-verify
//   bash scripts/ci/license_verify_tier.sh
//
package licensing
