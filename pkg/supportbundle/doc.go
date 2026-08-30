// Package supportbundle builds tar.gz diagnostic archives for ops support export.
//
// Role:
//   - Write streams gzip+tar: version.json, config/sanitized.env (keys only), goroutine.pprof, heap.pprof, logs/redacted.log, plus Options.ExtraJSON entries.
//   - RedactLine / RedactLog strip URLs, IPs, Stripe keys, and named secret JSON/kv fields before log inclusion.
//   - tailLogLines reads newest lines from *.log and *.json files under LogDir (defaultMaxLogLines cap per build).
//
// Defaults and limits:
//   - DefaultMaxBytes 50 MiB total archive (limitedWriter truncates and returns error when exceeded).
//   - DefaultTimeout 30s; callers wrap context (opsadmin, platformadmin) — Write checks ctx.Err() between sections only.
//   - defaultMaxLogLines 10000; LogDir defaults to LOGGER_DIR env, else /var/log/ad-event-processor.
//
// Topology:
//   - Invoked from internal/opsadmin POST /api/v1/ops/support/bundle via SupportBundleWriter.
//   - internal/controlplane supportBundleWriter supplies Meta, LogDir, client_rum.json ExtraJSON; platformadmin feedback attach reuses the same writer.
//   - Stdlib archive/tar/gzip and runtime/pprof only.
//
// Invariants:
//   - sanitized.env lists env keys with values replaced by ***; never writes raw secret values.
//   - RedactLog golden-tested; bundle build fails if archive exceeds MaxBytes after write.
//   - Missing log dir is non-fatal (empty logs/redacted.log section).
//
// Tradeoffs:
//   - Fixed-size bundle with redaction over full log upload: limits egress and support ticket secret leakage; not a full core dump.
//   - In-process pprof snapshot at build time only; no continuous profiling stream.
//
// Forbidden:
//   - Import internal/* packages.
//   - Include raw license private keys, DSN passwords, or unredacted IPs/URLs in tar members.
//
// Verify:
// go test ./pkg/supportbundle/... -short -run 'TestBundleRedactionGolden|TestWriteBundle' -count=1
package supportbundle
