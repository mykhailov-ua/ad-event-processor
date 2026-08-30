// Package netaddr normalizes listen addresses for gnet, stdlib dial/listen, and go-redis clients.
//
// Role:
//   - GnetListenURI prefixes tcp:// or unix:// when bare host:port or socket path is given.
//   - IsUnixSocketPath detects absolute paths, *.sock suffix, or paths containing .sock.
//   - DialTimeout, ListenUnix, PrepareUnixSocket, EnsureUnixSocketWritable for broker, tracker, and region-proxy sockets.
//   - ResolveListenAddr prefers unixPath over tcpAddr when unix path is non-empty.
//   - RedisUniversalOptions, RedisClientOptions, ParseRedisURL, RedisURLFromAddr unify TCP and unix Redis URLs.
//   - HTTPProbeTarget builds http://unix/health#<path> probe target for lifecycle health checks on UDS listeners.
//
// Topology:
//   - Hot: cmd/tracker wire (gnet unix listener), internal/broker, internal/regionproxy gnet bind.
//   - Cold: internal/database redis_connect, cmd/region-proxy, cmd/ivt-detector, pkg/lifecycle health probes.
//   - Depends on github.com/redis/go-redis/v9 for ParseRedisURL and option structs only.
//
// Invariants:
//   - GnetListenURI is idempotent when addr already has tcp:// or unix:// prefix.
//   - PrepareUnixSocket creates parent dir 0755 and removes stale socket file before ListenUnix.
//   - EnsureUnixSocketWritable requires ModeSocket; widens permissions to include group/other write (0o222) when missing.
//   - ParseRedisURL rejects empty input; unix:// and bare absolute socket paths dial with Network unix.
//   - RedisURLFromAddr embeds password and db in URL query for unix and redis:// forms.
//   - DialTimeout uses net.DialTimeout for TCP and net.Dialer with Timeout for unix.
//
// Tradeoffs:
//   - Heuristic IsUnixSocketPath (.sock suffix, contains .sock) vs strict absolute-only: supports relative redis.sock in dev compose.
//   - chmod toward 0o777 on UDS vs fail-closed: docker volume mounts often ship root-owned sockets; tracker retries EnsureUnixSocketWritable.
//   - Custom unix:// ParseRedisURL branch vs redis.ParseURL only: go-redis URL parser does not accept all appliance socket layouts.
//   - HTTPProbeTarget fragment encodes socket path: stdlib http client cannot dial unix without transport glue in lifecycle package.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/netaddr/... -short -count=1
//	go test ./pkg/netaddr/ -short -run TestGnetListenURI -count=1
//	go test ./pkg/netaddr/ -short -run TestEnsureUnixSocketWritable -count=1
package netaddr
