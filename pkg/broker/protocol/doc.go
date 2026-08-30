// Package protocol defines the broker wire codec and topic registry contracts.
//
// Role:
//   - Command opcodes: CmdProduce (1), CmdFetch (2), CmdProduceBatch (3), CmdRegisterTopic (4).
//   - Response opcodes: CmdProduceResp (101), CmdFetchResp (102), etc.
//   - TopicRegistry: in-memory ID map with optional file and Redis backing stores.
//   - ValidateConsumerGroup guards offset commit group names.
//
// Topology:
//   - Shared by pkg/broker/client, pkg/broker/log server side (internal/broker), and tests.
//   - FetchRespMetaLen=13; ProduceBatchRespMetaLen=13 bytes fixed header sizes.
//
// Defaults and limits:
//   - Topic IDs uint16; registry nextID atomic increment.
//   - CRC32 payload integrity on framed records (hash/crc32 IEEE).
//
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Changing opcode layout requires coordinated broker + client rollout.
//
// Verify:
// go test ./pkg/broker/protocol/... -short -count=1
package protocol
