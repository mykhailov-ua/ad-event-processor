// Package client is the TCP/unix wire client for the mmap WAL broker (Produce, Fetch, offsets).
//
// Role:
//   - Framed RPC over pkg/broker/protocol to cmd/broker / internal/broker gnet serve.
//   - Produce, ProduceBatch, Fetch, RegisterTopic, CommitOffset, CommittedOffset.
//   - Optional Redis URL for HA leader discovery (same pattern as log-shipper).
//   - MessageIterator walks fetch message blobs without per-message heap alloc.
//
// pkg/internal boundary:
//   - Callable from tracker (internal/stream/broker BrokerProducer), cmd/processor,
//     pkg/broker/consumer, pkg/regionproxy/client, cmd/log-shipper.
//   - Must not import internal/*; server logic stays in internal/broker.
//
// Topology:
//   - Dial via pkg/netaddr (tcp host:port or unix socket; runtimepaths.BrokerGnetSocket).
//   - Default tracker/processor timeouts set by callers (consumer package defaults 5s).
//   - Reused 1 MiB read/write buffers per Client; fetch iterator aliases response buffer.
//   - Leader lookup: Redis GET ad_event_processor:topics:{topic}/{partition}:leader then
//     ad_event_processor:brokers:{id} (requires SetRedisURL before Connect).
//
// Wire retry contract:
//   - Produce/Fetch retry up to 5 times with 500 ms backoff on conn errors.
//   - Broker response status 4-7 are retryable: 4 not leader, 5 stale fencing epoch,
//     6 leader catching up, 7 overloaded (admission shedding). Client closes conn and
//     re-resolves leader when RedisURL is set.
//   - CommittedOffset status OffsetStatusStoreUnavailable (2): fail-open, returns offset 0.
//
// Defaults and limits:
//   - NewClient timeout zero means no SetDeadline on dial/read/write.
//   - Produce batch size bounded by broker max message size (server flag).
//
// Tradeoffs:
//   - Synchronous Produce on client thread vs tracker BrokerProducer async ring (hot path
//     must enqueue, not block on fsync tiers).
//   - Fail-open CommittedOffset when store down vs fail-closed produce (resume may replay).
//   - Single shared fetchIter per Client: not safe for concurrent Fetch without external lock.
//
// Forbidden:
//   - pkg/* importing internal/*.
//   - Mock client unit tests as broker-primary cutover proof (fault tier required).
//
// Verify:
// go test ./pkg/broker/client/ -short -count=1
package client
