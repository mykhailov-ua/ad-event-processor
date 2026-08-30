// Package netaddr normalizes listen addresses for gnet and detects Redis Unix socket paths.
//
// Role:
//   - GnetListenURI prefixes tcp:// or unix:// when bare host:port or socket path given.
//   - IsUnixSocketPath and RedisDialOptions helpers shared by database redis_connect.
//
// Topology:
//   - Imported by internal/broker, internal/database, and tracker wire; stdlib + redis client options only.
//
// Invariants:
//   - Empty addr rejected at dial helpers; unix paths must be absolute or *.sock suffix.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/netaddr/... -short -count=1
package netaddr
