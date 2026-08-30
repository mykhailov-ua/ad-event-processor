// Package ingress defines the broker topic name for region-proxy WAL fan-in.
//
// Role:
//   - DefaultTopic constant region-proxy-ingress for pkg/regionproxy/client Produce calls.
//
// Topology:
//   - pkg/regionproxy/client wraps pkg/broker/client with topic registration once.
//
// Forbidden:
//   - internal/ingest must not import pkg/regionproxy.
//
// Verify:
// go test ./pkg/regionproxy/client/... -short -count=1
package ingress
