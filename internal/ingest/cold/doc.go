// Package cold hosts ingest-side matcher, attribution payload, and composite encoding helpers.
//
// Role:
//   - AppendAttributionPayload and JSON field append without encoding/json on hot paths.
//   - Composite fingerprint and metrics helpers shared by track and OpenRTB react glue.
//
// Topology:
//   - Pure functions over []byte and domain.Event; called from Tier B handler path after parse.
//   - Uses internal/ingest/parser AppendJSONString and internal/track SubID slots.
//
// Forbidden:
//   - Heap-heavy json.Marshal on synchronous /track accept path.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package cold
