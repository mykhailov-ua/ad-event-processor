// Package client produces regional spend-sync batches to the mmap WAL broker ingress topic.
//
// Role:
//   - Thin wrapper over pkg/broker/client: sync.Once RegisterTopic, then ProduceBatch per payload.
//   - ProduceSpendSyncPayload encodes one [][]byte row to ingress.DefaultTopic (region-proxy-ingress).
//   - Returns broker ProduceBatchResult (Committed count) for SyncWorker spend flush metrics.
//
// Topology:
//   - cmd/processor wires newRegionProxySpendSync when MULTI_REGION_ENABLED=1 and REGION_PROXY_ADDR set.
//   - Payload bytes are dedupkey.EncodeSpendSyncPayload JSON; region-proxy gnet appends them to WAL.
//   - Broker addr matches cmd/broker serve listen (tcp :9092 or unix socket); not the gnet region-proxy socket.
//   - Optional Config.RedisURL enables broker HA leader discovery on the underlying broker client.
//
// Defaults and limits:
//   - Config.Timeout default 5s when zero; passed to bclient.NewClient dial timeout.
//   - Topic registration errors surface from ensureTopic on first ProduceSpendSyncPayload call.
//   - Client does not open WAL, run keygen/opkey/uplink, or participate in quorum leases.
//
// Invariants:
//   - One Client per processor cell; RegisterTopic runs at most once per Client lifetime.
//   - Produce uses context.Background; caller owns payload encoding and batch sizing (GLOBAL_SPEND_BATCH_MIN).
//
// Forbidden:
//   - internal/ingest (non-_test) importing pkg/regionproxy/client on tracker hot path.
//   - Using ProduceSpendSyncPayload as proof of global PG spend apply (broker commit != control ingest).
//
// Verify:
// go test ./pkg/regionproxy/client/... -short -count=1
// go test ./internal/ingest/ -short -run SpendSync -count=1
package client
