// Package monotime exposes runtime.nanotime for hot-path deadline and metrics sampling.
//
// Role:
//   - Single //go:linkname site for ingest, filter, openrtb, rtb, and netintel callers.
//
// Forbidden:
//   - Wall-clock time for hot-path deadlines (use MonotonicNano / Nano only).
//
// Verify:
//
//	go test ./pkg/monotime/ -count=1
package monotime
