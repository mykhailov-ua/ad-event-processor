// Package main is the on-host install and lifecycle CLI entrypoint.
//
// Role:
//   - Delegate to internal/installer.CLI: preflight, provision, configure, up, bootstrap, apply, rollback, doctor, license.
//   - Host provisioning, compose stack bring-up, and license install/activate/status/host-id helpers.
//
// Topology:
//   - Operator-run shell tool; invokes docker compose, curl management API, and filesystem layout under repo root.
//   - main.go is thin; commands live in internal/installer.
//
// Invariants:
//   - Unknown subcommand prints usage and returns error exit 1 from main.
//   - apply supports --dry-run before mutating services.
//
// Forbidden:
//   - Not tracker/control runtime; no ingest or admin HTTP server in this binary.
//
// Verify:
// go run ./cmd/installer/
// go run ./cmd/installer/ preflight
// go test ./internal/installer/... -short -count=1
package main
