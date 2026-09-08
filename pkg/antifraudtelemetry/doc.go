// Package antifraudtelemetry scores compact client antifraud snapshots on the tracker hot path.
//
// Role:
//   - Pure Go scoring for temporal behavior, automation leaks, and RTT jitter heuristics.
//   - Called from internal/filter AntifraudTelemetryFilter after ingest parses JSON "antifraud".
//
// Invariants:
//   - No I/O, no Redis, no context.WithTimeout; must stay sub-microsecond on typical snapshots.
//   - Fail-open when snapshot fields are zero/absent; filter gates on AntifraudSet.
//
// Verify:
//
//	go test ./pkg/antifraudtelemetry/ -short -run TestScore -count=1
//	go test ./internal/ingest/ -short -run TestParseAntifraud -count=1
package antifraudtelemetry
