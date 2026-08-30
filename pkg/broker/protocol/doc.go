// Package protocol defines the broker wire codec, topic registry, and offset RPC payloads.
//
// Role:
//   - Length-prefixed TCP frames shared by pkg/broker/client and internal/broker server.
//   - Command codecs: Produce, Fetch, ProduceBatch, RegisterTopic, CommitOffset, CommittedOffset.
//   - TopicRegistry with optional FileRegistryStore (.topics/registry.json) and Redis registrar.
//   - ValidateConsumerGroup for offset commit group names.
//
// pkg/internal boundary:
//   - Stable wire contract in pkg/*; server handlers in internal/broker import this package only.
//   - Changing opcode or field layout requires coordinated broker + client rollout.
//
// Wire frame (after 4-byte BE total length):
//   - Body: cmd u16 + seq u64 + payload + crc32 u32 (CRC32-IEEE over payload only).
//   - Min body 14 bytes (empty payload). ReadFrameConn verifies checksum and command ID.
//
// Request commands:
//   - CmdProduce (1), CmdFetch (2), CmdProduceBatch (3), CmdRegisterTopic (4).
//   - CmdCommitOffset (5), CmdCommittedOffset (6).
//
// Response commands:
//   - CmdProduceResp (101), CmdFetchResp (102), CmdProduceBatchResp (103), CmdRegisterTopicResp (104).
//   - CmdCommitOffsetResp (105), CmdCommittedOffsetResp (106).
//
// Produce / fetch payload shapes:
//   - Produce request: topic_len u16 + topic + partition u16 + message bytes.
//   - Fetch request: topic + partition + start_offset u64 + max_bytes u32.
//   - Produce response meta: status u8 + assigned_offset u64 (9 bytes in payload).
//   - Fetch response meta (FetchRespMetaLen=13): status u8 + msg_count u32 + high_watermark u64;
//     trailing bytes are fetch blob for client MessageIterator.
//   - Fetch blob record: length u32 (8+payload) + offset u64 + payload (length includes offset).
//   - ProduceBatch: batch iterator over topic_id u16 + payload_len u32 + payload records.
//   - ProduceBatchResp meta (ProduceBatchRespMetaLen=13): status u8 + offset u64 + committed_count u32.
//
// Produce/fetch status bytes (server-defined, client retries 4-7):
//   - 0 OK; 1 malformed request; 2 topic/partition error; 3 append or disk error.
//   - 4 not leader; 5 stale fencing epoch; 6 leader catching up; 7 admission overloaded.
//
// Offset RPC:
//   - Key wire: topic_len u16 + topic + partition u16 + group_len u16 + group.
//   - Commit adds offset u64 after key. Response meta 9 bytes: status u8 + offset u64.
//   - OffsetStatusOK (0), OffsetStatusBadRequest (1), OffsetStatusStoreUnavailable (2).
//   - TopicPartitionID: "{topic}/{partition}" string key for server offset store maps.
//
// Topic registry:
//   - Topic IDs uint16; TopicRegistry nextID atomic; FileRegistryStore JSON snapshot version 1.
//   - Merge(clusterWins) for HA registry reconciliation.
//
// Defaults and limits:
//   - Topic and group names max 255 bytes; empty group rejected.
//
// Tradeoffs:
//   - CRC over payload only (not cmd/seq): cheaper hot path; length prefix bounds damage.
//   - uint16 topic IDs in batch path vs string topic in single produce (batch compactness).
//   - File registry vs Redis registrar: standalone broker vs HA cluster topic ID stability.
//
// Forbidden:
//   - pkg/* importing internal/*.
//   - Silent opcode or status enum drift without client/server joint release.
//
// Verify:
// go test ./pkg/broker/protocol/ -short -count=1
package protocol
