// Package ingest is tracker HTTP ingress: gnet server, handlers, filter wiring, OpenRTB/landing glue.
//
// Role:
//   - AdsPacketHandler React path for /track, /click, /openrtb/bid, /tg/*.
//   - processTrack calls FilterEngine.Check synchronously on PinnedWorkerPool Tier B (not a detached goroutine).
//   - tryAcquireStreamAdmission (TryReserve) before debit; publishAcceptedTrack after accept.
//
// Thread model (canonical: hot-path.mdc Tracker thread model, cmd/tracker/doc.go):
//
//	Tier A - gnet epoll (OnTraffic, WithLockOSThread(false)):
//	  - Peek/parse HTTP, PinParsedHTTPRequest, copy bytes to worker arena, SubmitOffloadToWorker, Discard frame.
//	  - Returns to epoll after enqueue; must not call FilterEngine.Check or synchronous Redis EVALSHA.
//
//	Tier B - PinnedWorkerPool worker (Worker.start, LockOSThread):
//	  - runOffloadedRequest -> React -> parseTrackIngest
//	    -> tryAcquireStreamAdmission (TryReserve)
//	    -> processTrack -> FilterEngine.Check (incl. EVALSHA when not local-quanta full-skip)
//	    -> publishAcceptedTrack -> writeGnetTrackAccepted -> cloneAsyncWriteBytes -> AsyncWrite.
//	  - Synchronous end-to-end on one worker; no go func() around FilterEngine.Check.
//	  - Per-worker MPSC queue depth 8192; HTTP1OffloadBusy = one in-flight offload per HTTP/1 conn.
//	  - Queue full -> WorkerPoolRejectTotal, 503 overload (TestFault_PinnedWorkerPoolSaturationSpike).
//
// Buffer lifetime:
//   - gnet peek frame: Tier A only until Discard.
//   - OffloadHTTPPin / worker arena: Tier B until releaseOffloadBuffers after response serialized.
//   - evt string fields may alias pin/arena during synchronous FilterEngine.Check on Tier B.
//   - Response bytes: cloneAsyncWriteBytes before arena release and AsyncWrite.
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
//   - Go-local filters before any Redis RTT; UnifiedFilter last (TestFilterEngine_ProductionOrder).
//   - FilterDeadlineMono uses monotonic ns; no wall clock in filter deadline loops.
//   - OpenRTB auction short-circuits before FilterEngine when RTB catalog handles the event.
//
// Contracts:
//
// Env knobs (units):
//   - FILTER_TIMEOUT_MS (ms): filter chain deadline; production ceiling 100 ms (config/env_validate.go).
//   - MAX_WORKERS: PinnedWorkerPool size (default 16); per-worker queue depth wired to 8192 in cmd/tracker/wire.go.
//   - STREAM_PRODUCER_ADMISSION_PCT (percent 0-100, default 85): TryReserve before Lua debit.
//   - MAX_REQUEST_BODY_SIZE (bytes, default 1048576): HTTP/1 maxBody passed to httpingress.ParseHTTP1.
//   - ORTB_SCAN_MAX_BYTES (bytes, default 262144): OpenRTB JSON scan cap (parser.OrtbScanMaxBytes).
//   - LOCAL_QUOTA_MODE: live enables local-quanta full-skip (zero sync EVALSHA when eligible).
//   - JSON_STRICT_UTF8 (bool, default true): parser strict UTF-8 rejection.
//   - PROTO_MAX_FIELDS (default 256): protobuf field budget before UnmarshalVT (filterwire).
//
// Defaults and limits:
//   - Worker MPSC queue: power-of-two depth; invalid config rounds to 4096 (gnet/worker.go).
//   - Worker arena: 4 slots x 1 MiB per worker; request pool cap 64 KiB before heap fallback.
//   - HTTP1 offload: one in-flight request per HTTP/1 connection (HTTP1OffloadBusy).
//   - Filter chain order (tracker wire): license, license_rps, emergency, geo, schedule, vpp, fraud,
//     residential, tcp_mss, device, l7_wire, json_serialization, behavior_telemetry, consent, segment,
//     entitlements, unified (TestFilterEngine_TrackerSegmentAfterLocalFilters).
//
// Tradeoffs:
//   - Tier A/B pinned workers vs detached goroutine for FilterEngine.Check:
//     Rejected go func() around filter: breaks pin/arena lifetime for evt string fields, adds scheduling
//     jitter, and complicates zero-alloc verification. Tier B runs Check synchronously on LockOSThread
//     worker with monotonic FILTER_TIMEOUT_MS deadline inside the same goroutine.
//   - Pin/arena copy vs holding full gnet peek frame through filter:
//     Tier A Discard frame immediately after enqueue; Tier B uses PinParsedHTTPRequest into OffloadHTTPPin
//     plus optional worker-arena wire copy. Rejected passing unsafe.String over the discarded peek buffer:
//     filter may read evt fields after frame release. Response path always cloneAsyncWriteBytes before
//     releaseOffloadBuffers.
//   - HTTP1OffloadBusy (one in-flight per conn) vs unbounded per-conn pipeline queue:
//     Rejected unbounded conn queue: slow filter would buffer unbounded requests and hide overload.
//     Busy bit stalls Tier A read loop until AsyncWrite completes; global backpressure is per-worker MPSC
//     depth 8192 with 503 on PushCtx failure.
//   - filterwire as wiring vs filter logic in ingest:
//     filterwire holds type aliases, constructors, BrandCreativeStore, and proto field-budget decode;
//     FilterEngine.Check and UnifiedFilter Lua live in internal/filter. Keeps ingest handler bundles
//     free of filter implementation while avoiding a second filter API surface.
//   - Stream admission before debit vs post-debit enqueue only:
//     TryReserve at STREAM_PRODUCER_ADMISSION_PCT rejects overload with 503 before spend; post-debit
//     failure requires budget-rollback.lua (TestUnifiedFilter_RollbackDebit_LocalQuanta).
//
// Forbidden:
//   - FilterEngine.Check or sync Redis on Tier A gnet epoll thread.
//   - go func() around FilterEngine.Check on /track.
//   - Postgres, ClickHouse, or outbox on synchronous /track accept path.
//   - Documenting filter as a detached goroutine or blocking Redis as forbidden on Tier B workers.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
//	go test ./internal/ingest/ -short -run TestPinnedWorkerPool -count=1
//	go test ./internal/ingest/ -short -run TestFault_PinnedWorkerPoolSaturationSpike -count=1
//	go test ./internal/ingest/ -short -run TestChaos_CrossHop_NginxGnet -count=1
//	make test-alloc-gate
package ingest
