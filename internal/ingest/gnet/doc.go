// Package gnet implements the tracker gnet HTTP ingress engine.
//
// Role:
//   - HTTP/1 DFA parse on epoll thread; optional offload to PinnedWorkerPool for /track-heavy paths.
//   - ConnContext pools, PinParsedHTTPRequest, arena-backed request copies, AsyncWrite response path.
//
// Thread model (see hot-path.mdc Tracker thread model, cmd/tracker/doc.go):
//
//	Tier A (OnTraffic, gnet event loop):
//	  - Peek/parse, SubmitOffloadToWorker, Discard frame; no FilterEngine or sync Redis.
//
//	Tier B (PinnedWorkerPool Worker.start, LockOSThread):
//	  - runOffloadedRequest -> Server.React; synchronous handler/filter path on same worker.
//
// Buffer lifetime:
//   - OffloadHTTPPin holds copied header/body slices for pinned parse path.
//   - Worker arena via submitOffloadToWorkerIdx; releaseOffloadBuffers after response serialized.
//   - cloneAsyncWriteBytes copies HTTP response before arena release and AsyncWrite.
//
// Queues:
//   - Per-worker MPSC queue (default depth from wire: 8192 per worker).
//   - HTTP1OffloadBusy: one in-flight offload per HTTP/1 connection.
//
// Forbidden:
//   - Synchronous Redis EVALSHA on Tier A epoll thread.
//   - Returning ConnContext to pool before AsyncWrite completes when OffloadAsyncWrite set.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run TestPinnedWorkerPool -count=1
//	go test ./internal/ingest/ -short -run TestFault_PinnedWorkerPoolSaturationSpike -count=1
package gnet
