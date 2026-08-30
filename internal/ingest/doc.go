// Package ingest is tracker HTTP ingress: gnet server, handlers, filter wiring, OpenRTB/landing glue.
//
// Role:
//   - AdsPacketHandler React path for /track, /click, /openrtb/bid, /tg/*.
//   - processTrack calls FilterEngine.Check synchronously on PinnedWorkerPool Tier B (not a detached goroutine).
//   - tryAcquireStreamAdmission (TryReserve) before debit; publishAcceptedTrack after accept.
//
// Thread model (hot-path.mdc, cmd/tracker/doc.go):
//   - Tier A (gnet OnTraffic, LockOSThread false): peek/parse, PinParsedHTTPRequest, SubmitOffloadToWorker, Discard; no sync Redis.
//   - Tier B (PinnedWorkerPool Worker.start, LockOSThread): runOffloadedRequest -> React -> processTrack -> FilterEngine.Check -> publish.
//
// Topology:
//   - Subpackages: gnet, httpingress, parser, conn, filterwire, cold, compat, ortbreact, watchers, pool, traceprobe.
//   - Filter types from internal/filter; ingest wiring via filterwire and compat re-exports.
//   - Must not import internal/controlplane admin or internal/fraud ML scoring.
//
// Invariants:
//   - At most one sync Redis EVALSHA per accept (zero when local-quanta full-skip eligible).
//   - TryReserve before Lua debit; RollbackDebit on post-debit enqueue failure.
//   - Zero heap allocs on /track parse path (make test-alloc-gate).
//
// Forbidden:
//   - FilterEngine.Check or sync Redis on Tier A gnet epoll thread.
//   - go func() around FilterEngine.Check on /track.
//   - Postgres, ClickHouse, or outbox on synchronous /track accept path.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
//	go test ./internal/ingest/ -short -run TestPinnedWorkerPool -count=1
//	go test ./internal/ingest/ -short -run TestChaos_CrossHop_NginxGnet -count=1
//	make test-alloc-gate
package ingest
