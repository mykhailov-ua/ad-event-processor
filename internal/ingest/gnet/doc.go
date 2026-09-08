// Package gnet implements the tracker gnet HTTP ingress engine.
//
// Role:
//   - HTTP/1 DFA and H2 ingress parse on epoll thread; optional offload to PinnedWorkerPool for /track-heavy paths.
//   - ConnContext pools, PinHTTP1RequestInPlace, arena-backed request copies, AsyncWrite response path.
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
// Invariants:
//   - ConnContext not returned to pool while OffloadAsyncWrite is set.
//   - HTTP1OffloadBusy cleared only after offload write completes (http1OffloadWriteDone).
//   - Conn WorkerID sticky across requests on same HTTP/1 or H2 connection (connWorkerAssign).
//   - MPSC queue PushCtx fails closed when write-read gap reaches capacity (no silent drop).
//
// Contracts:
//
// Env knobs (units; read via Server.cfg):
//   - MAX_WORKERS: worker count (default 16); WORKER_POOL_QUEUE_DEPTH default 8192 (cmd/tracker/wire.go).
//   - MAX_REQUEST_BODY_SIZE (bytes, default 1048576): ParseHTTP1 maxBody.
//   - HTTP1_INCOMPLETE_MAX (count, default 3): incomplete header/body strikes before close.
//   - HTTP1_BODY_IDLE_MS (ms): incomplete read idle close (header or body); dev default 500, production 5000 when unset.
//   - HTTP1_MAX_CONN_LIFETIME_MS (ms): hard conn cap from accept; production default 120000 when unset; 0 disables.
//   - HTTP1_MAX_PIPELINE_DEPTH: max complete pipelined HTTP/1 or multiplexed H2 requests per conn (production default 4).
//   - HTTP1_MAX_PIPELINE_BUSY_BYTES: inbound byte cap per conn while HTTP1OffloadBusy or before H2 parse (production default 256 KiB).
//   - HTTP1_TCP_USER_TIMEOUT_MS (ms): TCP_USER_TIMEOUT on inbound TCP conns; production default 15000; 0 disables.
//   - HTTP1_TCP_LINGER_SEC: SetLinger on accept; production default 0 (abortive close); -1 skips.
//   - HTTP1_MIN_CHUNKED_DATA_BYTES: OpenRTB chunked body floor; production default 64; <=1 disables.
//
// Defaults and limits:
//   - Per-worker MPSC queue depth WORKER_POOL_QUEUE_DEPTH (default 8192; must be power of two; zero or non-POT rounds to 4096).
//   - Worker arena: offloadArenaSlots=4, offloadMaxReqBytes=1 MiB per slot (arena.go).
//   - requestBufferPool objects capped at maxPoolObjectSize=64 KiB; larger wire copies heap-allocate.
//   - http1MaxBufferedOverhead=8192 bytes added to maxBody for inbound buffer accounting.
//   - ConnContext prealloc: BufSlice 4096, OffloadHTTPPin cap 2048, ChunkScratch cap 4096.
//
// Tradeoffs:
//   - Tier A epoll vs Tier B LockOSThread workers:
//     Epoll thread must stay non-blocking; sync Redis EVALSHA runs on pinned workers only. Rejected
//     running FilterEngine on epoll: blocks the whole connection accept loop.
//   - PinHTTP1RequestInPlace into OffloadHTTPPin vs re-parsing from arena wire copy:
//     Pin copies method/path/header/body slices into ConnContext.OffloadHTTPPin so React can skip
//     re-parse when OffloadReqPin is set. Tier A uses ParseHTTP1LimitsInto into OffloadReq before pin.
//     Fallback re-parse from OffloadReqSlice when pin unavailable.
//   - Worker arena (4 x 1 MiB fixed slots) vs sync.Pool vs heap:
//     Arena gives zero-alloc copy for typical requests; pool fallback when arena slots busy; heap when
//     wire len > 64 KiB or pool push fails. Rejected unbounded arena: bounded RAM per worker under flood.
//   - HTTP1OffloadBusy vs pipelining multiple offloads per connection:
//     One in-flight offload per HTTP/1 or H2 conn; Tier A breaks read loop while busy. Rejected per-conn
//     unbounded queue: would decouple overload signals from filter latency.
//   - H2 Tier B offload:
//     ParseH2Ingress on Tier A; PinHTTP1RequestInPlace into OffloadHTTPPin (no wire re-parse on worker).
//     ProtoH2 + H2StreamID on offload ConnContext for H2WrapH1Response on Tier B write path.
//   - HTTP/1 vs HTTP/2 pipeline caps:
//     HTTP1_MAX_PIPELINE_DEPTH and HTTP1_MAX_PIPELINE_BUSY_BYTES apply to both stacks (H1 depth via
//     CountCompleteHTTP1Messages; H2 via CountCompleteH2Messages in onTrafficH2).
//   - Bounded per-worker MPSC vs unbounded channel:
//     PushCtx returns false at depth 8192; caller returns 503 WorkerPoolRejectTotal. Rejected unbounded
//     channel: OOM under sustained overload instead of fail-closed 503.
//   - Sticky WorkerID vs round-robin per request:
//     Conn assigned WorkerID on first offload for cache warmth; SubmitOffloadToWorker falls back to
//     round-robin when target queue is full.
//   - AsyncWrite + cloneAsyncWriteBytes vs synchronous write on worker:
//     Response cloned to lease buffer; arena released before gnet callback frees lease. Rejected blocking
//     write on worker: holds arena slots and delays queue drain.
//
// Queues:
//   - Per-worker MPSC queue (default depth from wire: 8192 per worker).
//   - HTTP1OffloadBusy: one in-flight offload per HTTP/1 or H2 connection.
//
// Forbidden:
//   - Synchronous Redis EVALSHA on Tier A epoll thread.
//   - Returning ConnContext to pool before AsyncWrite completes when OffloadAsyncWrite set.
//
// Verify:
//
//	go test ./internal/ingest/ -short -run 'TestAdsPacketHandler_h2WorkerPool|TestPinnedWorkerPool' -count=1
//	go test ./internal/ingest/ -short -run TestFault_PinnedWorkerPoolSaturationSpike -count=1
package gnet
