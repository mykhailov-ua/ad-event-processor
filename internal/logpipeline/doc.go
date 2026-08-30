// Package logpipeline owns log shipper/evacuator/compactor shared types and chunk framing helpers.
//
// Role:
//   - Framing and retention metadata used by cmd/log-shipper, cmd/log-evacuator, and cmd/log-compactor.
//   - No admin HTTP; sidecar binaries only.
//
// Topology:
//   - Node-local I/O near mmap log segments; optional compose tools profile.
//
// Invariants:
//   - Chunk headers include checksum; corrupt chunk skipped with metric at caller.
//
// Forbidden:
//   - Hot-path tracker imports.
//
// Verify:
//
//	go test ./internal/logpipeline/... -short -count=1
package logpipeline
