// Package ingress defines the broker topic name for region-proxy WAL fan-in.
//
// Role:
//   - DefaultTopic constant region-proxy-ingress: single source of truth for pkg/regionproxy/client Produce calls.
//   - Same wire name as internal/regionproxy.DefaultIngressTopic on the gnet WAL server partition 0.
//
// Topology:
//   - Regional processor -> pkg/regionproxy/client -> broker topic region-proxy-ingress -> region-proxy coordinator fetch.
//   - Primary region-proxy registers the topic in protocol.TopicRegistry and wal.Partition append path.
//   - pkg/broker/log and internal/broker coordinator treat this topic like any other broker partition.
//
// Contracts:
//   - Topic string is stable across releases; changing it requires broker log migration and compose env updates.
//   - Partition id 0 only in current tree (protocol.TopicPartitionID(DefaultTopic, 0)).
//
// Forbidden:
//   - Duplicating the topic literal in new call sites; import ingress.DefaultTopic or server DefaultIngressTopic.
//   - internal/ingest hot path importing this package.
//
// Verify:
// go test ./pkg/regionproxy/client/... -short -count=1
// go test ./internal/regionproxy/ -short -run TestRegionProxy_ProduceBatchIngress -count=1
package ingress
