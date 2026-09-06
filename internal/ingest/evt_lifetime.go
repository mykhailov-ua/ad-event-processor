package ingest

// Tier B track event and buffer lifetime (hot-path.mdc, tradeoffs.mdc).
//
// gnet ConnContext:
//   - evt is &ConnContext.Evt for the duration of runOffloadedRequest on one PinnedWorkerPool worker.
//   - String fields may alias OffloadHTTPPin or worker arena bytes until releaseOffloadBuffers.
//   - FilterEngine.Check must run synchronously on that worker; no go func() around filter or publish setup.
//
// Response path:
//   - cloneAsyncWriteBytes before AsyncWrite; retireOffloadContext returns ConnContext to contextPool only after write callback.
//
// net/http /track:
//   - domain.EventPool Get must call Reset before reuse; Put on all terminal paths.
//
// Verify:
// bash scripts/ci/static/ingest_lifetime_gate.sh
// go test ./internal/ingest/ -short -run TestPublishAcceptedOrRollback_holdout -count=1
