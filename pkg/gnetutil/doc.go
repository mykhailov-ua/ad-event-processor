// Package gnetutil provides shared gnet connection idle and max-lifetime policies.
//
// Role:
//   - ConnPolicy: read-idle timeout and hard max connection lifetime (defaults below).
//   - ConnState: per-connection opened-at and read-idle deadline arming.
//   - OpenConn arms max-lifetime read deadline on accept.
//   - OnFrameProgress resets read-idle arming and slides max-lifetime deadline on each frame.
//   - WaitIncomplete arms read-idle deadline when a frame is incomplete; returns close reason.
//   - MaxLifetimeExceeded and ClearReadDeadline support broker/region-proxy close paths.
//
// Defaults and limits:
//   - DefaultConnReadIdle 30s (incomplete read with no further bytes).
//   - DefaultConnMaxLifetime 120s (hard cap from ConnState.OpenedAt).
//   - Zero duration in ConnPolicy falls back to defaults via ReadIdleDuration/MaxLifetimeDuration.
//
// Topology:
//   - Imported by internal/broker and internal/regionproxy gnet servers (conn_gnet.go wrappers).
//   - Not used by cmd/tracker ingest gnet; tracker has a separate epoll/pinned-worker model (hot-path.mdc).
//   - Metrics for close reason recorded by caller (BrokerConnIdleCloseTotal, RegionProxyConnIdleCloseTotal).
//
// Invariants:
//   - WaitIncomplete returns "" while connection stays open, or "read_idle" / "max_lifetime" to close.
//   - Read-idle deadline never extends past OpenedAt + MaxLifetime.
//   - EnsureConnState allocates ConnState once per connection; subsequent calls reuse context.
//
// Zero-alloc / performance:
//   - After ConnState is stored on gnet.Conn, OnFrameProgress, WaitIncomplete, and MaxLifetimeExceeded
//     are stack-only (no per-frame heap allocs).
//   - One heap allocation per new connection for *ConnState in EnsureConnState/NewConnState.
//   - SetReadDeadline syscalls are intentional; broker/region-proxy are cold-sidecar paths, not /track SLA.
//
// Fail-closed:
//   - Policy is fail-closed on connection hygiene: idle or lifetime exceeded yields close reason for caller.
//   - Does not fail-open partial frames indefinitely; read deadline forces teardown instead of hung peers.
//   - Nil ConnState in helpers is a no-op (caller must call OpenConn/EnsureConnState on accept).
//
// Tradeoffs:
//   - Shared package vs duplicating deadline logic in broker and region-proxy: one policy, two servers.
//   - Read deadline polling via gnet SetReadDeadline vs kernel TCP keepalive: explicit operator knobs.
//   - Not shared with tracker: ingest uses Tier A/B worker offload, not long-lived broker framing.
//
// Forbidden:
//   - Import internal/* packages.
//   - Use on cmd/tracker /track ingress without explicit hot-path review.
//
// Verify:
//   go test ./pkg/gnetutil/... -short -count=1
//   go test ./internal/broker/ -short -run TestFault_StaleLeaderFencingRejected -count=1
package gnetutil
