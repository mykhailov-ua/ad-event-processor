// Package main applies cold-path Postgres schema migrations from repo migration dirs.
//
// Role:
//   - Require DB_DSN; connect with database.Connect.
//   - Default: apply ads (internal/ingest/migrations), auth (internal/identity/migrations), billing (internal/ledger/migrations), notifier (notify.ApplyMigrations).
//   - -only comma filter: ads, auth, billing, notifier subsets.
//   - Tracked apply via pkg/coldpath.ApplyTrackedSchemaMigrations.
//
// Topology:
//   - One-shot CLI; must run from repository root (go.mod discoverable).
//   - Pool max 4 conns, min 1; no Redis or ClickHouse.
//
// Invariants:
//   - Missing DB_DSN exits 1 before connect.
//   - notifier set runs notify.ApplyMigrations only when selected.
//
// Forbidden:
//   - Not sqlc codegen (use make gen); does not migrate controlplane public schema beyond listed sets.
//
// Verify:
//
//	go list -e ./cmd/migrate-cold-path/
//	go test ./pkg/coldpath/ -short -run TestReadLimitedBody -count=1
//	DB_DSN=postgres://... go run ./cmd/migrate-cold-path/ -only billing,notifier
package main
