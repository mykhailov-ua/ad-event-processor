// Package verify implements Ed25519 JWT verify, HWID v2/v3 bind, install tokens, file MAC,
// activation enforcement, and optional release guard hooks.
//
// Role:
//   - verify.go / sign.go: JWT decode, Ed25519 signature validation, vendor signing helpers.
//   - hwid*.go: Linux telemetry collection and Argon2id HWID bind (P-HWID-01).
//   - file_verify.go / file_read.go / install_token.go: read and atomically write var/license.jwt.
//   - bind.go / activation*.go: deployment fingerprint and activation-limit checks on apply.
//   - key_resolver.go / public_key.go: kid-based public key resolution; decoy_* for red-team builds.
//   - guard*.go: optional ptrace/gdb watchdog on garbled control builds
//     (AD_EVENT_PROCESSOR_LICENSE_GUARD*, licensing.mdc).
//   - file_mac.go / mck_stretch.go: enterprise license file MAC and recheck stretch.
//
// Topology:
//   - licensingadmin Service.ApplyLicenseToken, cmd/license-issue, internal/licensing watcher reload.
//   - entitlements_aliases.go re-exports entitlements helpers for property and red-team tests.
//
// Invariants:
//   - Invalid signature or malformed JWT rejects apply (P-C2-01).
//   - IngestAllowed false only for EXPIRED or REVOKED (P-C3-03; properties_test.go).
//   - Effective() customer limits never exceed deployment ceiling (P-C4-03; properties_test.go).
//   - JWT hwid_hash when present takes precedence over legacy host fingerprint (bind.go).
//
// Forbidden:
//   - Per-event JWT crypto on /track (inline licensing snapshot only on hot path).
//
// Verify:
//
//	go test ./internal/licensing/verify/ -short -run 'TestVerifyJWT|TestProperty_P_C|TestVerifyDeploymentBind|TestInstallToken' -count=1
//	make license-red-team
//	make license-verify
package verify
