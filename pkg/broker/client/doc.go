// Package client is the gnet broker wire client for Produce and Fetch.
//
// Role:
//   - TCP or unix socket to cmd/broker serve (default 127.0.0.1:9092 or runtimepaths.BrokerGnetSocket).
//   - CmdProduce, CmdFetch, CmdProduceBatch, CmdRegisterTopic over pkg/broker/protocol framing.
//   - Optional Redis URL for HA leader discovery (same pattern as log-shipper).
//
// Topology:
//   - Called from tracker BrokerProducer, cmd/log-shipper, pkg/broker/consumer, pkg/regionproxy/client.
//   - MessageIterator walks fetch response batches without per-message alloc on hot read path.
//
// Defaults and limits:
//   - Default dial timeout 5s when unset on NewClient.
//   - Produce batch size capped by broker server max message size.
//
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Mock client in unit tests is not broker-primary cutover proof (fault tier required).
//
// Verify:
// go test ./pkg/broker/client/... -short -count=1
// go test ./pkg/broker/... -short -count=1
package client
