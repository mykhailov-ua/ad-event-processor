// Package clickhouse contains report SQL builders and row mapping helpers (no HTTP).
//
// Role:
//   - Typed query fragments for placements, geo-device, spend velocity, RTB overview, and related reports.
//   - Dimension parsing and safe math helpers shared by reports handlers.
//
// Invariants:
//   - Queries run through database.ClickHouseQuery with readonly DSN in production reads.
//   - No PG writes from this package.
//
// Verify:
//
//	go test ./internal/reports/clickhouse/ -short -count=1
package clickhouse
