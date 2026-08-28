//   Package rtb implements the in-process OpenRTB auction engine (catalog scan,
//   RunAuction, shadow/live modes). Called from ingestion on exchange routes.
//
//   Verify:
//     go test ./internal/rtb/ -short -count=1
//     go test ./internal/rtb/ -run='^$' -bench=BenchmarkAuction -benchmem -count=1
//
// Budget: RunAuction p99 < 15 us; catalog scan p99 < 500 candidates (core.mdc).
//
package rtb
