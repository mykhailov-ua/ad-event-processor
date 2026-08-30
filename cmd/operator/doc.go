// Package main is the customer-facing operator Cobra CLI.
//
// Role:
//   - Cobra root with --env-path; loads .env before subcommands.
//   - doctor: stack health checks against running compose services.
//   - doctor bundle: export support bundle via internal/doctor.
//   - doctor --checklist: print MVSS checklist rows from DATA_SECURITY.md (no network I/O).
//
// Topology:
//   - Reads internal/config where commands need DSN or service URLs.
//   - No long-running server; one-shot HTTP/FS checks against running stack.
//
// Forbidden:
//   - Not vendor license-issue or trial-registry (vendor plane binaries).
//   - Not a replacement for Grafana/Prometheus incident dashboards.
//
// Verify:
// go run ./cmd/operator/ --help
// go run ./cmd/operator/ doctor --checklist
// go test ./cmd/operator/... -short -count=1
package main
