// Package main is the tracker hot-path binary: gnet HTTP ingress for /track, /click,
// /openrtb/bid, /tg/*, and static track.js.
//
// Role:
//   - Accept or reject under end-to-end SLA; 202/302 does not imply PG/CH wrote.
//   - Run FilterEngine (local gates then UnifiedFilter Redis Lua) off the gnet epoll loop.
//   - Enqueue accepted events via StreamProducer (Redis XADD) or BrokerProducer (mmap WAL).
//   - Serve campaign catalog from atomic.Pointer snapshot; reload via Redis pub/sub and PG boot sync.
//
// Topology:
//   - gnet event loops (WithLockOSThread(false)) + PinnedWorkerPool (runtime.LockOSThread per worker); not net/http on ingest.
//   - Multi-shard Redis (one sync EVALSHA max per accept when not local-quanta full-skip).
//   - PG read pool: boot-time registry/slot-map/settings fallback only; never on request thread.
//   - HTTP: SERVER_PORT default 8181 (compose lanes 8181-8184 per TRACKER_INSTANCE).
//   - Metrics sidecar: METRICS_PORT default 9090 (compose 9101-9104); /metrics, /health, /ready, optional pprof.
//
// Thread model (see hot-path.mdc Tracker thread model):
//
//	Tier A - gnet epoll (OnTraffic):
//	  - Peek/parse HTTP frame, PinParsedHTTPRequest + copy raw bytes to worker arena, SubmitOffloadToWorker, Discard frame.
//	  - Returns to epoll immediately; must not call FilterEngine.Check or synchronous Redis EVALSHA.
//
//	Tier B - PinnedWorkerPool worker (runOffloadedRequest -> React):
//	  - parseTrackIngest, tryAcquireStreamAdmission, processTrack -> FilterEngine.Check (incl. sync EVALSHA),
//	    publishAcceptedTrack, serialize response.
//	  - Synchronous end-to-end on the same LockOSThread worker; no go func() around FilterEngine.Check.
//	  - Per-worker MPSC queue depth 8192 (MAX_WORKERS queues); HTTP1OffloadBusy = one in-flight offload per HTTP/1 conn.
//	  - Queue full -> WorkerPoolRejectTotal, 503 worker-pool overload (TestFault_PinnedWorkerPoolSaturationSpike).
//
// wire.go init order (runTracker):
//  1. Retry policy, runtime autotune, logger + 15s metrics reporter.
//  2. License guard; PG pool + registry bootstrap (Sync, StartSync, optional replica file).
//  3. Redis shards, pool warm (REDIS_POOL_SIZE + MAX_WORKERS pings), license epoch sync.
//  4. StaticSlotSharder + slot map PG load; SlotMapWatcher goroutine (SLOT_MAP_POLL_INTERVAL_MS).
//  5. Budget cache warm; registry pub/sub + epoch poll; optional broker campaign-update watcher.
//  6. Consent store watch; GeoIP (+ hot-reload watcher GEOIP_WATCHER_INTERVAL_SEC).
//  7. Local filter chain pieces; SettingsWatcher (1s tick goroutine).
//  8. Stream trimmer; InitUnifiedFilterLua + PreloadScripts + script preheater (30s).
//  9. Optional local quanta (LOCAL_QUOTA_MODE shadow|live); FilterEngine assembly.
//  10. Optional RTB catalog sync; NewAdsPacketHandler; optional PG failover subscriber.
//  11. StreamProducer or BrokerProducer (CH_INGEST_SOURCE=broker); fraud backpressure watcher.
//  12. Optional UDP/TCP ingress control; L1 intel tables on handler; PinnedWorkerPool (queue 8192 per worker).
//  13. gnet.Run; METRICS_PORT HTTP sidecar; signal drain (gnet, workers, quanta, broker, registry).
//
// SLA (core.mdc):
//   - ad_http_request_duration_seconds p95 < 50 ms; p99 target 80 ms; hard ceiling 100 ms.
//   - FILTER_TIMEOUT_MS production <= 100 ms (dev default 5000 ms); monotonic deadline on pinned worker only.
//
// Invariants:
//   - No synchronous Postgres or ClickHouse on gnet epoll or pinned worker request path.
//   - No ML inference (internal/fraud scoring); fraud boost is atomic snapshot only.
//   - At most one sync Redis EVALSHA per accept (zero when local quanta full-skip eligible).
//   - TryReserve before Lua debit; rollback on post-debit enqueue failure.
//   - Zero heap allocs on /track parse path (make test-alloc-gate).
//
// Layout constants (internal/ingest, internal/filter, internal/stream):
//
//	Generational snapshots (atomic.Value):
//	  - Registry campaignMapSnapshot: full map swap on sync/pubsub/epoch reload; snapGen invalidates per-worker cache.
//	  - StaticSlotSharder SlotMapSnapshot: PG version gate in SlotMapWatcher; GetShard is zero-lock read.
//	  - SettingsWatcher / fraud boost maps: atomic snapshot readers on filter path.
//
//	TTL and stale drivers:
//	  - Registry stale mode: REGISTRY_STALE_TTL_SEC since last pub/sub OK (fail-open settings PG fallback).
//	  - Duplicate/idempotency: DUPLICATE_TTL_SEC, IDEMPOTENCY_TTL_HRS on Redis and local quanta idem cache.
//	  - Registry epoch poll: REGISTRY_POLL_MS compares Redis campaign:registry:epoch across shards.
//
//	Worker-local cache:
//	  - registryWorkerCacheSlot: one campaign per pinned worker index, keyed by snapGen (lazy reload on miss).
//	  - PinnedWorkerPool routes conn WorkerID to same core; arena reuses offload request buffers.
//
//	Buffer lifetime (internal/ingest/gnet):
//	  - PinParsedHTTPRequest copies header/body slices into ConnContext.OffloadHTTPPin.
//	  - SubmitOffload copies raw frame into worker arena when possible.
//	  - evt string fields may reference OffloadHTTPPin/arena until pinned worker handler returns.
//	  - Accept/Origin copied via string() before response path; response via cloneAsyncWriteBytes then releaseOffloadBuffers.
//
//	StreamProducer TryReserve vs local quanta:
//	  - TryReserve: reserves slot on StreamProducer or BrokerProducer ring before debit (fail-closed overload).
//	  - Local quanta full-skip: Go ledger debit + LocalQuantaStreamPublisher async lane (Redis fcap/budget side effects);
//	    does not bypass TryReserve for deferred CH_INGEST_SOURCE=broker or Redis stream primary path.
//	  - Post-debit enqueue fail: budget-rollback.lua or local quanta refund; metric ad_stream_producer_post_debit_rejected_total.
//
// Forbidden:
//   - Import internal/controlplane admin or internal/fraud ML scoring.
//   - fmt.Sprintf, interface{}, context.With* on hot request path (CI static gates).
//   - Synchronous Redis EVALSHA on gnet epoll event-loop threads.
//   - Spawning go func() around FilterEngine.Check on /track (breaks buffer lifetime and zero-alloc contract).
//
//   - FilterEngine.Check runs synchronously on PinnedWorkerPool Tier B, not on gnet epoll threads.
//   - gnet epoll returns after SubmitOffload; pinned workers run the full accept path before picking next job.
//
// Env defaults:
//   - SERVER_PORT, METRICS_PORT (TCP ports).
//   - FILTER_TIMEOUT_MS (ms), FILTER_SLOW_MS (ms), WRITE_TIMEOUT_MS (ms).
//   - MAX_WORKERS (pinned filter workers), REGISTRY_SYNC_INTERVAL_MS (ms), REGISTRY_POLL_MS (ms).
//   - REDIS_POOL_SIZE (connections per shard), STREAM_PRODUCER_ADMISSION_PCT (percent 0-100).
//   - TRACKER_PPROF_ENABLED=1 enables /debug/pprof on metrics port (write timeout 120s vs 10s).
//
// Verify:
// go test ./internal/ingest/ -short -count=1
// make test-alloc-gate
// go test ./internal/ingest/ -run TestFault_ -count=1
// bash scripts/test/load/gate_run.sh
// go test ./cmd/tracker/ -run TestMainReady -count=1
package main
