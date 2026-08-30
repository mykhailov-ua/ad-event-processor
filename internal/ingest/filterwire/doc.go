// Package filterwire wires FilterEngine construction, protobuf eval wire, and ingest-side filter glue.
//
// Role:
//   - Type aliases and constructors bridging internal/filter and internal/filter/unified into ingest handlers.
//   - Proto field-budget wire decode for filter eval paths; re-exports filter sentinel errors.
//
// Topology:
//   - FilterEngine.Check runs on caller goroutine (tracker: PinnedWorkerPool Tier B, sync, not detached).
//   - UnifiedFilter last in chain; at most one EVALSHA per accept when not local-quanta full-skip.
//
// Forbidden:
//   - Postgres or ClickHouse queries inside synchronous Check from this wiring layer.
//   - internal/fraud ML scoring import; boost snapshot only via filter package.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestFilterEngine -count=1
//	go test ./internal/filter/... -short -count=1
package filterwire
