// Package supportbundle builds tar.gz diagnostic archives for ops support export.
//
// Role:
//   - bundle.go collects redacted config snippets, log tails, and optional pprof profiles.
//   - redact.go strips secrets; logs.go caps line count per file.
//
// Defaults and limits:
//   - DefaultMaxBytes 50 MiB archive size.
//   - DefaultTimeout 30s total build time.
//   - defaultMaxLogLines 10000 per log file.
//
// Topology:
//   - Invoked from opsadmin support bundle handler; stdlib archive/tar/gzip only.
//
// Forbidden:
//   - Import internal/* packages.
//   - Include raw license private keys or DSN passwords (redact layer).
//
// Verify:
//
//	go test ./pkg/supportbundle/... -short -count=1
package supportbundle
