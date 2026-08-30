// Package client produces region-proxy ingress batches to the mmap WAL broker topic.
//
// Role:
//   - Thin wrapper over pkg/broker/client with sync.Once topic registration.
//   - Default topic pkg/regionproxy/ingress.DefaultTopic (region-proxy-ingress).
//   - Optional RedisURL for broker HA leader discovery.
//
// Topology:
//   - Used by internal/regionproxy server and regional processor tests.
//   - Broker addr matches cmd/broker serve listen (9092 or unix socket).
//
// Defaults and limits:
//   - Dial timeout default 5s when Config.Timeout zero.
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy.
//
// Verify:
// go test ./pkg/regionproxy/client/... -short -count=1
package client
