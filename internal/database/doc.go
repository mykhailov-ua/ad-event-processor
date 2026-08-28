// Package database: Postgres pool, ClickHouse client facades, and shared DB helpers
// for cold path. Hot path uses Redis/Lua via ingestion, not this package, for filters.
//
package database
