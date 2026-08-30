// Package main post-processes vtproto output for ingest protobuf parse paths.
//
// Role:
//   - Default path internal/ingest/pb/events_vtproto.pb.go (override with argv[1]).
//   - Replace ExtraKeys/ExtraValues append+copy with appendReuseBytes(...) call sites.
//   - Strip nil-slice-to-empty-byte-slice guards via regex (reduces alloc on hot parse path).
//   - No-op when already patched; exit 1 if expected buf plugin patterns missing.
//
// Topology:
//   - Build-time text patch after buf generate; no runtime server.
//
// Invariants:
//   - Idempotent: second run exits 0 without write when patterns already applied.
//   - Failure when vtproto generator output shape changes (pattern missing message).
//
// Forbidden:
//   - Do not hand-edit generated pb.go except through this tool or regen workflow.
//   - Zero alloc on /track is enforced by make test-alloc-gate, not by this comment.
//
// Verify:
// make proto
// go run ./cmd/patch-vtproto-hotpath
// make test-alloc-gate
package main
