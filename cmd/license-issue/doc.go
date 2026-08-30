// Package main issues Ed25519-signed deployment license JWTs (vendor plane).
//
// Role:
//   - Parse flags in issue.go: SKU, customer, deployment_id, hwid, trial registry hooks, revoke, approve-pending.
//   - Sign JWT with AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE (vendor key).
//   - Enforce trial-registry anchors (telegram, hwid, usdt_tx) unless VENDOR_TRIAL_FORCE.
//   - Write token to --out or stdout via writeIssueOutput.
//
// Topology:
//   - Offline vendor CLI; uses internal/licensing and internal/trialregistry.
//   - main.go delegates to runIssue in issue.go.
//
// Invariants:
//   - exitUsage (2) on flag/help errors; 1 on signing, registry, or write failures.
//
// Forbidden:
//   - Not customer-side license apply (use admin Settings or POST /api/v1/license/apply on control).
//   - Never commit private key material or issued JWTs to the repo.
//
// Verify:
// go test ./cmd/license-issue/... -short -count=1
// make license-verify
package main
