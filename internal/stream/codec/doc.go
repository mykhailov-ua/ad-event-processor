// Package codec hosts sync.Pool-backed protobuf and byte-buffer helpers for stream enqueue.
//
// Role:
//   - StreamEventPool, AdLogRecordPool: vtproto message reuse on hot encode/decode paths.
//   - ByteBufPool, LogBufPool: reusable []byte backing for MarshalToSizedBufferVT output.
//   - ByteSliceValuePool, ProducerValuesPool: Redis XADD field wrappers without per-event heap alloc.
//   - ClearAdStreamEvent / DeepResetAdStreamEvent / DeepResetAdDLQEvent: return pooled messages to clean state.
//   - UnsafeBytes / UnsafeString: zero-copy string<->[]byte at arena lifetime boundaries on Tier B worker.
//   - MicroUnitFactor aliases pkg/money.MicroUnit for spend fields in stream payloads.
//
// Topology:
//   - Used by stream producer, broker producer/consumer payload parse, fraud stream writer, and auditlog on tracker Tier B.
//   - SliceToMap is cold-path helper for string-slice to set conversion (admin/config loaders only).
//
// Invariants:
//   - Callers must Clear/DeepReset pooled pb messages before Put; DeepReset retains slice capacity for next Get.
//   - UnsafeString/UnsafeBytes valid only while source bytes/string remain immutable for Tier B handler lifetime.
//   - Marshal output in ByteBufPool is handed to async queue/goroutine only after pb fields copied or buffer ownership transferred.
//   - Pooled *pb.AdStreamEvent must not alias domain.Event string fields past releaseOffloadBuffers on the pinned worker.
//
// Tradeoffs:
//   - sync.Pool over per-event heap alloc: StreamEventPool + ByteBufPool target 0 allocs/op on stream/broker encode;
//     DeepResetAdStreamEvent clears fields but keeps backing slice cap to avoid reallocate on next MarshalToSizedBufferVT.
//     Rejected: encoding/json or fresh []byte per field on hot enqueue path.
//   - UnsafeBytes on domain.Event strings: pbEvt fields alias evt.ClickID/Type/IP/UA during synchronous encode on Tier B;
//     valid until handler returns and arena/pin released. Rejected: holding unsafe views into gnet peek frame (discarded before filter).
//   - ByteSliceValue for Redis XADD: wraps []byte without string conversion alloc; broker path copies into ring slots instead.
//   - Async handoff rule: after enqueue to StreamProducer chan or broker ring, only the vtproto bytes buffer travels async;
//     pooled message returned to StreamEventPool immediately after Marshal (producer.go pattern). Holding pooled pb past goroutine
//     handoff without copy would race the next Get().
//
// Forbidden:
//   - encoding/json builders for production stream or broker payloads.
//   - Holding pooled *pb.AdStreamEvent past async goroutine handoff without copy.
//
// Verify:
//
//	go test ./internal/stream/ -short -run TestStreamPayloadSizeComparison -count=1
//	go test ./internal/stream/broker/ -short -run TestParseBrokerPayload_AdStreamEvent -count=1
//	go test ./internal/stream/broker/ -short -run TestParseBrokerPayloadStream_RawVTProto -count=1
package codec
