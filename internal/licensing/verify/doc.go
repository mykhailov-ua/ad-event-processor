// Package verify implements Ed25519 JWT verify, HWID v2/v3 binding, install tokens, and release guard hooks.
//
// Role:
//   - verify.go and sign.go validate license JWT claims; hwid*.go collect Linux telemetry for Argon2id bind.
//   - guard*.go optional ptrace/gdb watchdog on garbled control builds (licensing.mdc kill switches).
//
// Topology:
//   - Used by licensingadmin apply, cmd/license-issue, and control inline snapshot reload.
//   - file_verify.go reads var/license.jwt from configured path.
//
// Invariants:
//   - Invalid signature or payload rejects apply (P-C2-01).
//   - HWID match optional when JWT omits hwid_hash (legacy fingerprint path).
//
// Forbidden:
//   - Verify on every /track event (snapshot only on hot path).
//
// Verify:
//
//	go test ./internal/licensing/verify/... -short -count=1
//	make license-red-team
package verify
